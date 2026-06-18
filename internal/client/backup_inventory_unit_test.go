package client

import (
	"context"
	"net/http"
	"testing"
)

// These tests pin the two inventory waiters (backup virtual disk + virtual
// machine). They share the same shape: ANY read error is fatal at once (no
// transient retry — #293 Finding F1, pinned NOT fixed here), a nil result keeps
// polling (bounded by the injected backoff / context), and a found object returns.

type bvdOutcome struct {
	disk *BackupVirtualDisk
	err  error
}

func scriptedBackupVirtualDiskReads(calls *int, outcomes ...bvdOutcome) backupVirtualDiskReadFunc {
	return func(ctx context.Context) (*BackupVirtualDisk, error) {
		i := *calls
		if i >= len(outcomes) {
			i = len(outcomes) - 1
		}
		*calls++
		return outcomes[i].disk, outcomes[i].err
	}
}

type bvmOutcome struct {
	vm  *BackupVirtualMachine
	err error
}

func scriptedBackupVirtualMachineReads(calls *int, outcomes ...bvmOutcome) backupVirtualMachineReadFunc {
	return func(ctx context.Context) (*BackupVirtualMachine, error) {
		i := *calls
		if i >= len(outcomes) {
			i = len(outcomes) - 1
		}
		*calls++
		return outcomes[i].vm, outcomes[i].err
	}
}

func TestWaitForBackupVirtualDiskInventory(t *testing.T) {
	t.Run("a transient 5xx read error is FATAL at once (F1: no transient retry)", func(t *testing.T) {
		calls := 0
		read := scriptedBackupVirtualDiskReads(&calls, bvdOutcome{err: StatusError{Code: http.StatusInternalServerError}})
		_, err := waitForBackupVirtualDiskInventory(context.Background(), "disk-1", read, immediateBackoff(20), nil)
		if err == nil {
			t.Fatal("the current behavior treats ANY read error as fatal; a transient 5xx must fail (pins F1)")
		}
		if calls != 1 {
			t.Fatalf("calls=%d, want exactly 1 (no retry, even on a transient 5xx)", calls)
		}
	})

	t.Run("nil then found: waits then returns", func(t *testing.T) {
		calls := 0
		read := scriptedBackupVirtualDiskReads(&calls,
			bvdOutcome{}, // not yet in inventory
			bvdOutcome{disk: &BackupVirtualDisk{}},
		)
		disk, err := waitForBackupVirtualDiskInventory(context.Background(), "disk-1", read, immediateBackoff(20), nil)
		if err != nil || disk == nil {
			t.Fatalf("a nil result must keep polling until the disk appears, got err=%v", err)
		}
		if calls != 2 {
			t.Fatalf("calls=%d, want 2", calls)
		}
	})

	t.Run("stuck-nil is bounded by the injected backoff", func(t *testing.T) {
		calls := 0
		read := scriptedBackupVirtualDiskReads(&calls, bvdOutcome{}) // nil forever
		_, err := waitForBackupVirtualDiskInventory(context.Background(), "disk-1", read, immediateBackoff(3), nil)
		if err == nil {
			t.Fatal("a never-appearing disk must be bounded by the backoff/context, not loop forever")
		}
		if calls != 4 {
			t.Fatalf("calls=%d, want 4 (initial + 3 retries)", calls)
		}
	})

	t.Run("context cancellation stops polling", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		read := func(c context.Context) (*BackupVirtualDisk, error) {
			calls++
			cancel()
			return nil, nil
		}
		_, err := waitForBackupVirtualDiskInventory(ctx, "disk-1", read, immediateBackoff(50), nil)
		if err == nil {
			t.Fatal("context cancellation must stop the polling")
		}
		if calls != 1 {
			t.Fatalf("calls=%d, want 1 (the loop must honour the retry ctx)", calls)
		}
	})

	t.Run("found returns immediately", func(t *testing.T) {
		calls := 0
		read := scriptedBackupVirtualDiskReads(&calls, bvdOutcome{disk: &BackupVirtualDisk{}})
		disk, err := waitForBackupVirtualDiskInventory(context.Background(), "disk-1", read, immediateBackoff(20), nil)
		if err != nil || disk == nil {
			t.Fatalf("a found disk must return at once, got err=%v", err)
		}
		if calls != 1 {
			t.Fatalf("calls=%d, want 1", calls)
		}
	})
}

func TestWaitForBackupVirtualMachineInventory(t *testing.T) {
	t.Run("a transient 5xx read error is FATAL at once (F1: no transient retry)", func(t *testing.T) {
		calls := 0
		read := scriptedBackupVirtualMachineReads(&calls, bvmOutcome{err: StatusError{Code: http.StatusInternalServerError}})
		_, err := waitForBackupVirtualMachineInventory(context.Background(), "vm-1", read, immediateBackoff(20), nil)
		if err == nil {
			t.Fatal("the current behavior treats ANY read error as fatal; a transient 5xx must fail (pins F1)")
		}
		if calls != 1 {
			t.Fatalf("calls=%d, want exactly 1 (no retry, even on a transient 5xx)", calls)
		}
	})

	t.Run("nil then found: waits then returns", func(t *testing.T) {
		calls := 0
		read := scriptedBackupVirtualMachineReads(&calls,
			bvmOutcome{},
			bvmOutcome{vm: &BackupVirtualMachine{}},
		)
		vm, err := waitForBackupVirtualMachineInventory(context.Background(), "vm-1", read, immediateBackoff(20), nil)
		if err != nil || vm == nil {
			t.Fatalf("a nil result must keep polling until the VM appears, got err=%v", err)
		}
		if calls != 2 {
			t.Fatalf("calls=%d, want 2", calls)
		}
	})

	t.Run("stuck-nil is bounded by the injected backoff", func(t *testing.T) {
		calls := 0
		read := scriptedBackupVirtualMachineReads(&calls, bvmOutcome{})
		_, err := waitForBackupVirtualMachineInventory(context.Background(), "vm-1", read, immediateBackoff(3), nil)
		if err == nil {
			t.Fatal("a never-appearing VM must be bounded by the backoff/context, not loop forever")
		}
		if calls != 4 {
			t.Fatalf("calls=%d, want 4 (initial + 3 retries)", calls)
		}
	})

	t.Run("context cancellation stops polling", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		read := func(c context.Context) (*BackupVirtualMachine, error) {
			calls++
			cancel()
			return nil, nil
		}
		_, err := waitForBackupVirtualMachineInventory(ctx, "vm-1", read, immediateBackoff(50), nil)
		if err == nil {
			t.Fatal("context cancellation must stop the polling")
		}
		if calls != 1 {
			t.Fatalf("calls=%d, want 1 (the loop must honour the retry ctx)", calls)
		}
	})

	t.Run("found returns immediately", func(t *testing.T) {
		calls := 0
		read := scriptedBackupVirtualMachineReads(&calls, bvmOutcome{vm: &BackupVirtualMachine{}})
		vm, err := waitForBackupVirtualMachineInventory(context.Background(), "vm-1", read, immediateBackoff(20), nil)
		if err != nil || vm == nil {
			t.Fatalf("a found VM must return at once, got err=%v", err)
		}
		if calls != 1 {
			t.Fatalf("calls=%d, want 1", calls)
		}
	})
}
