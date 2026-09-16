package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// A single "not found" read-back used to be treated as proof that the platform had
// rolled the creation back: nothing was recorded and the operator was told to re-run
// `terraform apply`.
//
// That is unsound on this platform. The activity publishes the virtual machine item
// as soon as the object is materialised — about eight seconds after the POST — while
// GET-by-id has documented eventual-consistency windows right after a write (#415).
// Read-back and indexing can cross: the object exists and the id still answers 404.
// Acting on that single answer produces the very incident this file exists to
// prevent — an unattended billable virtual machine, plus a second one created by the
// advised re-apply.
//
// These tests pin that a not-found is only believed once repeated read-backs agree,
// and that anything contradicting the absence wins.
func withFastAbsenceBackoff(t *testing.T) {
	t.Helper()
	previous := vmAbsenceBackoff
	vmAbsenceBackoff = time.Millisecond
	t.Cleanup(func() { vmAbsenceBackoff = previous })
}

func TestConfirmVMAbsenceContradictions(t *testing.T) {
	withFastAbsenceBackoff(t)
	ctx := context.Background()
	const candidate = "vm-1"

	t.Run("a virtual machine appearing on a later read is NOT absent", func(t *testing.T) {
		calls := 0
		read := func(context.Context, string) (*client.VirtualMachine, error) {
			calls++
			// The indexing window closes between the first and the second read.
			return &client.VirtualMachine{ID: candidate, Name: "vm-under-create"}, nil
		}
		vm, err := confirmVMAbsence(ctx, read, candidate)
		if err != nil || vm == nil {
			t.Fatalf("a virtual machine that appears on re-read must be reported present, got vm=%v err=%v", vm, err)
		}
		if calls != 1 {
			t.Fatalf("expected the confirmation to stop at the first contradiction, got %d reads", calls)
		}
	})

	t.Run("an error on a later read is inconclusive, never absence", func(t *testing.T) {
		read := func(context.Context, string) (*client.VirtualMachine, error) {
			return nil, errors.New("gateway timeout")
		}
		vm, err := confirmVMAbsence(ctx, read, candidate)
		if err == nil {
			t.Fatal("a failing read proves nothing and must surface as an error, so the caller adopts instead of orphaning")
		}
		if vm != nil {
			t.Fatalf("expected no virtual machine alongside the error, got %v", vm)
		}
	})

	t.Run("a cancelled context is inconclusive, never absence", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(context.Background())
		cancel()
		read := func(context.Context, string) (*client.VirtualMachine, error) {
			t.Fatal("the read must not run once the context is cancelled")
			return nil, nil
		}
		if _, err := confirmVMAbsence(cancelled, read, candidate); err == nil {
			t.Fatal("a cancelled context proves nothing about the platform and must not be read as absence")
		}
	})

	t.Run("only a run of not-founds is believed", func(t *testing.T) {
		calls := 0
		read := func(context.Context, string) (*client.VirtualMachine, error) { calls++; return nil, nil }
		vm, err := confirmVMAbsence(ctx, read, candidate)
		if vm != nil || err != nil {
			t.Fatalf("expected confirmed absence, got vm=%v err=%v", vm, err)
		}
		if calls != vmAbsenceAttempts {
			t.Fatalf("expected %d confirmation reads, got %d", vmAbsenceAttempts, calls)
		}
	})
}

// TestRecoverVMCreateFailureRereadsBeforeConcludingAbsence covers the whole recovery,
// not just the helper: a first not-found followed by the virtual machine appearing
// must ADOPT it, which is the difference between tracking the machine and leaving it
// billable and unattended.
func TestRecoverVMCreateFailureRereadsBeforeConcludingAbsence(t *testing.T) {
	withFastAbsenceBackoff(t)
	const candidate = "vm-late"

	calls := 0
	read := func(context.Context, string) (*client.VirtualMachine, error) {
		calls++
		if calls == 1 {
			return nil, nil // the indexing window has not closed yet
		}
		return &client.VirtualMachine{ID: candidate, Name: "vm-under-create"}, nil
	}

	d := newVMCreateData(t)
	diags := recoverVMCreateFailure(context.Background(), d, read, runningActivity(vmItem(candidate)),
		"act-1", "vm-under-create", "failed to create virtual machine", errors.New("boom"))

	if !diags.HasError() {
		t.Fatal("the create failure must still be reported as an error")
	}
	if d.Id() != candidate {
		t.Fatalf("the virtual machine appeared on re-read and MUST be adopted, got id=%q. "+
			"Left unadopted it is orphaned: billable, outside the state, and the operator is "+
			"sent to re-apply — the #527 incident", d.Id())
	}
}

// TestRecoverVMCreateFailureAbsenceAdviceDoesNotSayReapply pins the diagnostic itself.
// "re-run terraform apply" is the one instruction that can manufacture a duplicate if
// the reads crossed an indexing window, so the confirmed-absence branch must tell the
// operator to CONFIRM first, like every other branch that records nothing.
func TestRecoverVMCreateFailureAbsenceAdviceDoesNotSayReapply(t *testing.T) {
	withFastAbsenceBackoff(t)
	read := func(context.Context, string) (*client.VirtualMachine, error) { return nil, nil }

	d := newVMCreateData(t)
	diags := recoverVMCreateFailure(context.Background(), d, read, runningActivity(vmItem("vm-gone")),
		"act-9", "vm-under-create", "failed to create virtual machine", errors.New("boom"))

	if !diags.HasError() {
		t.Fatal("expected an error")
	}
	if d.Id() != "" {
		t.Fatalf("a confirmed absence must record nothing, got id=%q", d.Id())
	}
	msg := diags[0].Summary
	if strings.Contains(msg, "re-run `terraform apply`") {
		t.Fatalf("the diagnostic must not send the operator straight back to apply: a crossed "+
			"indexing window would then leave an orphan AND create a second virtual machine.\ngot: %s", msg)
	}
	for _, want := range []string{"read-backs", "before re-applying", "ORPHANED", "vm-under-create"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("the diagnostic is missing %q, which the operator needs to act.\ngot: %s", want, msg)
		}
	}
}
