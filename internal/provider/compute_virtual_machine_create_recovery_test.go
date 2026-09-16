package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// newVMCreateData builds a ResourceData for a create in flight: the create
// inputs are seeded but NO id is set yet, exactly as when the deployment
// request has been accepted and the wait has not resolved an id.
func newVMCreateData(t *testing.T) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourceVirtualMachine().Schema, map[string]interface{}{})
	if err := d.Set("name", "vm-under-create"); err != nil {
		t.Fatalf("seeding name: %v", err)
	}
	return d
}

// readFatalVM is a "must not be reached" sentinel: it fails the test if the
// recovery reads the API on a path where no candidate should have been derived.
func readFatalVM(t *testing.T) vmReadFunc {
	return func(ctx context.Context, id string) (*client.VirtualMachine, error) {
		t.Fatalf("the read must not be reached, it was called with %q", id)
		return nil, nil
	}
}

func vmItem(id string) client.ActivityConcernedItem {
	return client.ActivityConcernedItem{ID: id, Type: "virtual_machine"}
}

func otherItem(id, typ string) client.ActivityConcernedItem {
	return client.ActivityConcernedItem{ID: id, Type: typ}
}

// runningActivity reproduces the shape measured against the API while a
// marketplace deployment is in flight: a single "running" state with NO result,
// and the concerned items the platform publishes as the objects materialise.
func runningActivity(items ...client.ActivityConcernedItem) *client.Activity {
	return &client.Activity{
		ID:             "activity-1",
		ConcernedItems: items,
		State:          map[string]client.ActivityState{"running": {Progression: 0.4}},
	}
}

// TestVMCandidateFromActivity pins the adoption evidence rule. An adopted id is
// persisted as a TAINTED resource, so the next apply DESTROYS whatever it
// points at: ambiguity must never resolve into an adoption.
func TestVMCandidateFromActivity(t *testing.T) {
	for _, tc := range []struct {
		name     string
		activity *client.Activity
		exclude  []string
		want     string
	}{
		{
			name:     "a nil activity yields no candidate",
			activity: nil,
			want:     "",
		},
		{
			name:     "no concerned item at all yields no candidate",
			activity: runningActivity(),
			want:     "",
		},
		{
			// The shape measured at submission time, before the VM exists.
			name: "only non-VM concerned items yields no candidate",
			activity: runningActivity(
				otherItem("item-1", "marketplace_item"),
				otherItem("clu-1", "host_cluster"),
				otherItem("dc-1", "datacenter"),
				otherItem("ds-1", "datastore"),
			),
			want: "",
		},
		{
			// The shape measured ~8s later, while STILL running: this is the
			// window in which the reported failure leaves the VM orphaned.
			name: "a single VM among other items is the candidate",
			activity: runningActivity(
				otherItem("item-1", "marketplace_item"),
				vmItem("vm-new"),
				otherItem("ds-1", "datastore"),
			),
			want: "vm-new",
		},
		{
			name:     "an empty VM id is never adopted",
			activity: runningActivity(vmItem("")),
			want:     "",
		},
		{
			// Defence in depth: an activity shape this code cannot interpret
			// must not be guessed at.
			name:     "two distinct VMs yield no candidate",
			activity: runningActivity(vmItem("vm-a"), vmItem("vm-b")),
			want:     "",
		},
		{
			name:     "the same VM listed twice is still a single candidate",
			activity: runningActivity(vmItem("vm-a"), vmItem("vm-a")),
			want:     "vm-a",
		},
		{
			// The clone path: whether or not the platform lists the source, the
			// exclusion makes the answer correct.
			name:     "the clone source is excluded and the new VM wins",
			activity: runningActivity(vmItem("vm-source"), vmItem("vm-clone")),
			exclude:  []string{"vm-source"},
			want:     "vm-clone",
		},
		{
			// The destructive case this rule exists for: if the activity only
			// ever referenced the source, adopting it would taint — and on the
			// next apply DESTROY — the source virtual machine.
			name:     "an activity naming only the clone source yields no candidate",
			activity: runningActivity(vmItem("vm-source")),
			exclude:  []string{"vm-source"},
			want:     "",
		},
		{
			name:     "an empty exclusion entry excludes nothing",
			activity: runningActivity(vmItem("vm-new")),
			exclude:  []string{""},
			want:     "vm-new",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := vmCandidateFromActivity(tc.activity, tc.exclude...)
			if got != tc.want {
				t.Fatalf("vmCandidateFromActivity = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRecoverVMCreateFailure pins the state decision taken on the create
// failure path. The worst outcome is an ORPHAN — a virtual machine that exists
// platform-side and is absent from the Terraform state, unreachable by
// `terraform destroy`. The second worst is a WRONG adoption, which taints (and
// therefore destroys) an object this resource never created.
//
// A mutant that reverts the failure path to `diag.Errorf(...)` without recovery
// reds on every case that expects an id.
func TestRecoverVMCreateFailure(t *testing.T) {
	ctx := context.Background()
	cause := errors.New("an error occured while getting the status of activity: i/o timeout")

	t.Run("an already-resolved id is kept and reported as tracked", func(t *testing.T) {
		d := newVMCreateData(t)
		d.SetId("vm-known")

		diags := recoverVMCreateFailure(ctx, d, readFatalVM(t), runningActivity(vmItem("vm-other")),
			"activity-1", "vm-under-create", "failed to deploy marketplace item", cause)

		if !diags.HasError() {
			t.Fatal("the underlying failure must still surface as an error")
		}
		if d.Id() != "vm-known" {
			t.Fatalf("a resolved id must never be replaced, got %q", d.Id())
		}
		requireContains(t, diags[0].Summary, "IS recorded in the Terraform state")
	})

	t.Run("no candidate records nothing and tells the operator what to audit", func(t *testing.T) {
		d := newVMCreateData(t)

		diags := recoverVMCreateFailure(ctx, d, readFatalVM(t),
			runningActivity(otherItem("item-1", "marketplace_item")),
			"activity-1", "vm-under-create", "failed to deploy marketplace item", cause)

		if !diags.HasError() {
			t.Fatal("the failure must surface as an error")
		}
		if d.Id() != "" {
			t.Fatalf("nothing may be adopted without evidence, got id %q", d.Id())
		}
		requireContains(t, diags[0].Summary, "ORPHANED")
		requireContains(t, diags[0].Summary, "vm-under-create")
		requireContains(t, diags[0].Summary, "activity-1")
	})

	t.Run("a confirmed VM is adopted and its machine manager recorded", func(t *testing.T) {
		d := newVMCreateData(t)
		read := func(ctx context.Context, id string) (*client.VirtualMachine, error) {
			if id != "vm-new" {
				t.Fatalf("the read must target the candidate, got %q", id)
			}
			return &client.VirtualMachine{
				ID:             "vm-new",
				Name:           "vm-under-create",
				MachineManager: client.BaseObject{ID: "mm-1", Name: "vc-001"},
			}, nil
		}

		diags := recoverVMCreateFailure(ctx, d, read, runningActivity(vmItem("vm-new")),
			"activity-1", "vm-under-create", "failed to deploy marketplace item", cause)

		if !diags.HasError() {
			t.Fatal("the failure must still surface as an error, the VM is tainted not healthy")
		}
		if d.Id() != "vm-new" {
			t.Fatalf("the confirmed VM must be adopted, got id %q", d.Id())
		}
		// Without machine_manager_id the read fails closed on a later refresh,
		// which would leave the operator unable to destroy the tainted VM.
		if got := d.Get("machine_manager_id").(string); got != "mm-1" {
			t.Fatalf("machine_manager_id must be recorded, got %q", got)
		}
		requireContains(t, diags[0].Summary, "HAS been recorded in the Terraform state")
	})

	t.Run("a definitive 404 records nothing rather than poisoning the state", func(t *testing.T) {
		// The absence is now confirmed by repeated read-backs (see confirmVMAbsence);
		// drive them without sleeping.
		withFastAbsenceBackoff(t)
		d := newVMCreateData(t)
		read := func(ctx context.Context, id string) (*client.VirtualMachine, error) {
			return nil, nil // client contract: (nil, nil) is a definitive absence
		}

		diags := recoverVMCreateFailure(ctx, d, read, runningActivity(vmItem("vm-gone")),
			"activity-1", "vm-under-create", "failed to deploy marketplace item", cause)

		if !diags.HasError() {
			t.Fatal("the failure must surface as an error")
		}
		if d.Id() != "" {
			t.Fatalf("an id that no longer resolves must NOT be adopted, got %q", d.Id())
		}
		requireContains(t, diags[0].Summary, "rolled the creation back")
	})

	// A marketplace or content-library deployment clones a vCenter template. If
	// the activity ever designated that template instead of the new virtual
	// machine, adopting it would taint the template and the next apply would
	// DESTROY it, breaking every deployment that sources it.
	t.Run("a template is never adopted", func(t *testing.T) {
		d := newVMCreateData(t)
		read := func(ctx context.Context, id string) (*client.VirtualMachine, error) {
			return &client.VirtualMachine{ID: "vm-template", Name: "vm-under-create", Template: true}, nil
		}

		diags := recoverVMCreateFailure(ctx, d, read, runningActivity(vmItem("vm-template")),
			"activity-1", "vm-under-create", "failed to deploy marketplace item", cause)

		if !diags.HasError() {
			t.Fatal("the failure must surface as an error")
		}
		if d.Id() != "" {
			t.Fatalf("a template must NEVER be adopted, got id %q", d.Id())
		}
		requireContains(t, diags[0].Summary, "TEMPLATE")
	})

	// The created virtual machine carries the requested name verbatim (measured
	// against the API). A mismatch means the activity designated another object.
	t.Run("a name mismatch is not adopted", func(t *testing.T) {
		d := newVMCreateData(t)
		read := func(ctx context.Context, id string) (*client.VirtualMachine, error) {
			return &client.VirtualMachine{ID: "vm-other", Name: "somebody-elses-vm"}, nil
		}

		diags := recoverVMCreateFailure(ctx, d, read, runningActivity(vmItem("vm-other")),
			"activity-1", "vm-under-create", "failed to deploy marketplace item", cause)

		if !diags.HasError() {
			t.Fatal("the failure must surface as an error")
		}
		if d.Id() != "" {
			t.Fatalf("a virtual machine carrying another name must NOT be adopted, got id %q", d.Id())
		}
		requireContains(t, diags[0].Summary, "somebody-elses-vm")
	})

	t.Run("an inconclusive read still adopts rather than orphan", func(t *testing.T) {
		d := newVMCreateData(t)
		read := func(ctx context.Context, id string) (*client.VirtualMachine, error) {
			return nil, errors.New("503 Service Unavailable")
		}

		diags := recoverVMCreateFailure(ctx, d, read, runningActivity(vmItem("vm-new")),
			"activity-1", "vm-under-create", "failed to deploy marketplace item", cause)

		if !diags.HasError() {
			t.Fatal("the failure must surface as an error")
		}
		if d.Id() != "vm-new" {
			t.Fatalf("a read that proves nothing must never orphan the VM, got id %q", d.Id())
		}
		requireContains(t, diags[0].Summary, "read-back was inconclusive")
	})

	t.Run("the clone source is never adopted", func(t *testing.T) {
		d := newVMCreateData(t)

		diags := recoverVMCreateFailure(ctx, d, readFatalVM(t), runningActivity(vmItem("vm-source")),
			"activity-1", "vm-under-create", "failed to clone virtual machine", cause, "vm-source")

		if !diags.HasError() {
			t.Fatal("the failure must surface as an error")
		}
		if d.Id() != "" {
			t.Fatalf("adopting the clone source would destroy it on the next apply, got id %q", d.Id())
		}
	})

	t.Run("a nil activity records nothing", func(t *testing.T) {
		d := newVMCreateData(t)

		diags := recoverVMCreateFailure(ctx, d, readFatalVM(t), nil,
			"activity-1", "vm-under-create", "failed to deploy marketplace item", cause)

		if !diags.HasError() {
			t.Fatal("the failure must surface as an error")
		}
		if d.Id() != "" {
			t.Fatalf("nothing may be adopted from a nil activity, got id %q", d.Id())
		}
		requireContains(t, diags[0].Summary, "ORPHANED")
	})
}

func requireContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("diagnostic %q must contain %q", got, want)
	}
}
