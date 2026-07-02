package provider

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func newVMInstanceRD(t *testing.T, raw map[string]interface{}) *schema.ResourceData {
	t.Helper()
	return schema.TestResourceDataRaw(t, resourcePublicCloudVMInstance().Schema, raw)
}

func vmiCompletedActivity(concernedID, result string) *client.Activity {
	a := &client.Activity{State: map[string]client.ActivityState{"completed": {Result: result}}}
	if concernedID != "" {
		a.ConcernedItems = []client.ActivityConcernedItem{{ID: concernedID, Type: "vmi"}}
	}
	return a
}

// okListPrimaryDisk is the listDisks seam for the found read path (which now
// enriches os_disk from the primary disk).
func okListPrimaryDisk(ctx context.Context, id string) ([]*client.PublicCloudVMDisk, error) {
	return []*client.PublicCloudVMDisk{{ID: "osdisk-1", Position: 0, SizeGb: 38, StorageType: "st-1", IsPrimary: true}}, nil
}

// --- resize precondition (CustomizeDiff logic) ---

func TestVMInstanceResizeRequiresOff(t *testing.T) {
	cases := []struct {
		name     string
		exists   bool
		resizing bool
		power    string
		wantErr  bool
	}{
		{"create with power on and cpu set is allowed (boot at create)", false, true, "on", false},
		{"existing resize while on is refused", true, true, "on", true},
		{"existing resize while off is allowed", true, true, "off", false},
		{"existing non-resize while on is allowed", true, false, "on", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := vmInstanceResizeRequiresOff(tc.exists, tc.resizing, tc.power)
			if (err != nil) != tc.wantErr {
				t.Fatalf("exists=%v resizing=%v power=%q -> err=%v, wantErr=%v", tc.exists, tc.resizing, tc.power, err, tc.wantErr)
			}
		})
	}
}

// --- update ordering (pure plan) ---

func TestPlanVMInstanceUpdate(t *testing.T) {
	cases := []struct {
		name            string
		metadataChanged bool
		resizing        bool
		osDiskExtending bool
		oldPS, newPS    string
		want            []vmUpdateOp
	}{
		{"metadata only", true, false, false, "off", "off", []vmUpdateOp{vmOpPatch}},
		{"stop before resize (on->off + resize)", false, true, false, "on", "off", []vmUpdateOp{vmOpStop, vmOpResize}},
		{"start last (off->on)", false, false, false, "off", "on", []vmUpdateOp{vmOpStart}},
		{"resize while already off", false, true, false, "off", "off", []vmUpdateOp{vmOpResize}},
		{"full: metadata + stop + resize", true, true, false, "on", "off", []vmUpdateOp{vmOpPatch, vmOpStop, vmOpResize}},
		{"no change", false, false, false, "off", "off", nil},
		{"stop only", false, false, false, "on", "off", []vmUpdateOp{vmOpStop}},
		{"os_disk extend after resize (both, while off)", false, true, true, "off", "off", []vmUpdateOp{vmOpResize, vmOpExtendOSDisk}},
		{"os_disk extend between resize and start", false, true, true, "off", "on", []vmUpdateOp{vmOpResize, vmOpExtendOSDisk, vmOpStart}},
		{"stop then os_disk extend (on->off + extend, no resize)", false, false, true, "on", "off", []vmUpdateOp{vmOpStop, vmOpExtendOSDisk}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := planVMInstanceUpdate(tc.metadataChanged, tc.resizing, tc.osDiskExtending, tc.oldPS, tc.newPS)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("plan = %v, want %v", got, tc.want)
			}
			// Structural invariants: stop precedes resize and the os_disk extend;
			// both write ops (resize, extend) precede a start; start is last.
			stopIdx := indexOfVMOp(got, vmOpStop)
			resizeIdx := indexOfVMOp(got, vmOpResize)
			extendIdx := indexOfVMOp(got, vmOpExtendOSDisk)
			startIdx := indexOfVMOp(got, vmOpStart)
			if stopIdx >= 0 && resizeIdx >= 0 && stopIdx > resizeIdx {
				t.Fatalf("stop must precede resize, got %v", got)
			}
			if stopIdx >= 0 && extendIdx >= 0 && stopIdx > extendIdx {
				t.Fatalf("stop must precede the os_disk extend, got %v", got)
			}
			if resizeIdx >= 0 && extendIdx >= 0 && resizeIdx > extendIdx {
				t.Fatalf("resize must precede the os_disk extend, got %v", got)
			}
			if startIdx >= 0 {
				if startIdx != len(got)-1 {
					t.Fatalf("start must be the last op, got %v", got)
				}
				if extendIdx >= 0 && extendIdx > startIdx {
					t.Fatalf("the os_disk extend must precede a start, got %v", got)
				}
			}
		})
	}
}

func indexOfVMOp(ops []vmUpdateOp, op vmUpdateOp) int {
	for i, o := range ops {
		if o == op {
			return i
		}
	}
	return -1
}

// --- update execution order (recording seams) ---

func TestExecuteVMInstanceUpdate(t *testing.T) {
	t.Run("executes ops in the given order", func(t *testing.T) {
		var order []string
		wait := func(ctx context.Context, id string) (*client.Activity, error) {
			return vmiCompletedActivity("", "vm-1"), nil
		}
		funcs := vmInstanceCRUDFuncs{
			patch: func(ctx context.Context, id string, r *client.PatchVMInstanceRequest) (string, error) {
				order = append(order, "patch")
				return "a", nil
			},
			stop: func(ctx context.Context, id string) (string, error) { order = append(order, "stop"); return "a", nil },
			resize: func(ctx context.Context, id string, r *client.ResizeVMInstanceRequest) (string, error) {
				order = append(order, "resize")
				return "a", nil
			},
			extendSystem: func(ctx context.Context, id string, size int) (string, error) {
				order = append(order, "extend")
				return "a", nil
			},
			start:        func(ctx context.Context, id string) (string, error) { order = append(order, "start"); return "a", nil },
			waitActivity: wait,
		}
		plan := []vmUpdateOp{vmOpPatch, vmOpStop, vmOpResize, vmOpExtendOSDisk, vmOpStart}
		if err := executeVMInstanceUpdate(context.Background(), "vm-1", plan, funcs, &client.PatchVMInstanceRequest{}, &client.ResizeVMInstanceRequest{}, 50); err != nil {
			t.Fatalf("execute: %v", err)
		}
		want := []string{"patch", "stop", "resize", "extend", "start"}
		if !reflect.DeepEqual(order, want) {
			t.Fatalf("call order = %v, want %v", order, want)
		}
	})

	t.Run("a failing op stops execution and wraps the op name", func(t *testing.T) {
		var order []string
		wait := func(ctx context.Context, id string) (*client.Activity, error) {
			return vmiCompletedActivity("", "vm-1"), nil
		}
		funcs := vmInstanceCRUDFuncs{
			stop: func(ctx context.Context, id string) (string, error) {
				order = append(order, "stop")
				return "", errors.New("boom")
			},
			resize: func(ctx context.Context, id string, r *client.ResizeVMInstanceRequest) (string, error) {
				order = append(order, "resize")
				return "a", nil
			},
			waitActivity: wait,
		}
		err := executeVMInstanceUpdate(context.Background(), "vm-1", []vmUpdateOp{vmOpStop, vmOpResize}, funcs, nil, &client.ResizeVMInstanceRequest{}, 0)
		if err == nil {
			t.Fatal("a failing op must return an error")
		}
		if len(order) != 1 || order[0] != "stop" {
			t.Fatalf("execution must stop at the failing op, got %v", order)
		}
	})
}

// --- create orchestration ---

func createRD(t *testing.T) *schema.ResourceData {
	return newVMInstanceRD(t, map[string]interface{}{
		"name":                 "web",
		"availability_zone_id": "11111111-1111-1111-1111-111111111111",
		"template_id":          "22222222-2222-2222-2222-222222222222",
		"instance_family_id":   "33333333-3333-3333-3333-333333333333",
		"cpu":                  2,
		"memory":               4,
		"backup_policy_id":     "44444444-4444-4444-4444-444444444444",
		"power_state":          "on",
		"os_network_adapter": []interface{}{
			map[string]interface{}{"device_index": 0, "network_id": "55555555-5555-5555-5555-555555555555"},
		},
	})
}

func TestCreateVMInstanceWith(t *testing.T) {
	okRead := func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error) {
		return &client.PublicCloudVMInstance{ID: id, Name: "web", Status: "running", VCPU: 2, RAMGb: 4}, nil
	}

	t.Run("id comes from the activity concernedItems (vmi)", func(t *testing.T) {
		d := createRD(t)
		funcs := vmInstanceCRUDFuncs{
			create: func(ctx context.Context, r *client.CreateVMInstanceRequest) (string, error) { return "act-1", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return vmiCompletedActivity("vm-1", "vm-1"), nil
			},
			read:      okRead,
			listDisks: okListPrimaryDisk,
		}
		if diags := createVMInstanceWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Id() != "vm-1" {
			t.Fatalf("id = %q, want vm-1", d.Id())
		}
		if d.Get("status").(string) != "running" || d.Get("power_state").(string) != "on" {
			t.Fatalf("state not populated: status=%q power=%q", d.Get("status"), d.Get("power_state"))
		}
	})

	t.Run("id falls back to the terminal state result when no vmi concernedItem", func(t *testing.T) {
		d := createRD(t)
		funcs := vmInstanceCRUDFuncs{
			create: func(ctx context.Context, r *client.CreateVMInstanceRequest) (string, error) { return "act-1", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return vmiCompletedActivity("", "vm-2"), nil
			},
			read:      okRead,
			listDisks: okListPrimaryDisk,
		}
		if diags := createVMInstanceWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Id() != "vm-2" {
			t.Fatalf("id = %q, want vm-2 (from state result)", d.Id())
		}
	})

	t.Run("no id reported fails closed without setting an id", func(t *testing.T) {
		d := createRD(t)
		funcs := vmInstanceCRUDFuncs{
			create: func(ctx context.Context, r *client.CreateVMInstanceRequest) (string, error) { return "act-1", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return vmiCompletedActivity("", ""), nil
			},
		}
		diags := createVMInstanceWith(context.Background(), d, funcs)
		if !diags.HasError() {
			t.Fatal("an activity without a VM id must fail")
		}
		if d.Id() != "" {
			t.Fatalf("no id must be set when the activity reports none, got %q", d.Id())
		}
	})

	t.Run("a wait failure fails and never sets an id", func(t *testing.T) {
		d := createRD(t)
		funcs := vmInstanceCRUDFuncs{
			create: func(ctx context.Context, r *client.CreateVMInstanceRequest) (string, error) { return "act-1", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nil, errors.New("activity failed")
			},
		}
		diags := createVMInstanceWith(context.Background(), d, funcs)
		if !diags.HasError() {
			t.Fatal("a wait failure must fail the create")
		}
		if d.Id() != "" {
			t.Fatalf("a wait failure must not set an id, got %q", d.Id())
		}
	})

	t.Run("a create error fails and never sets an id", func(t *testing.T) {
		d := createRD(t)
		funcs := vmInstanceCRUDFuncs{
			create: func(ctx context.Context, r *client.CreateVMInstanceRequest) (string, error) {
				return "", errors.New("bad request")
			},
		}
		diags := createVMInstanceWith(context.Background(), d, funcs)
		if !diags.HasError() {
			t.Fatal("a create error must fail")
		}
		if d.Id() != "" {
			t.Fatalf("a create error must not set an id, got %q", d.Id())
		}
	})
}

// --- read state-safety (E0-9) ---

func TestReadVMInstanceInto(t *testing.T) {
	vm := &client.PublicCloudVMInstance{ID: "vm-1", Name: "web", Status: "stopped", VCPU: 1, RAMGb: 2, DisksSizeGb: 38}

	t.Run("found populates state and maps power_state from status", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{
			read:      func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error) { return vm, nil },
			listDisks: okListPrimaryDisk,
		}
		if diags := readVMInstanceInto(context.Background(), d, funcs, vmInstanceReadForRefresh); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Get("power_state").(string) != "off" {
			t.Fatalf("stopped status must map to power_state off, got %q", d.Get("power_state"))
		}
		if d.Get("disks_size_gb").(int) != 38 {
			t.Fatalf("disks_size_gb not set: %d", d.Get("disks_size_gb"))
		}
	})

	t.Run("read error keeps the resource and errors", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{read: func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error) {
			return nil, errors.New("403")
		}}
		diags := readVMInstanceInto(context.Background(), d, funcs, vmInstanceReadForRefresh)
		if !diags.HasError() {
			t.Fatal("a read error must fail closed")
		}
		if d.Id() != "vm-1" {
			t.Fatal("a read error must not drop the resource")
		}
	})

	t.Run("afterWrite nil keeps the id and errors (never orphan)", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{read: func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error) { return nil, nil }}
		diags := readVMInstanceInto(context.Background(), d, funcs, vmInstanceReadAfterWrite)
		if !diags.HasError() {
			t.Fatal("a nil read right after a write must fail closed, not drop")
		}
		if d.Id() != "vm-1" {
			t.Fatal("afterWrite must never drop the fresh id")
		}
	})

	t.Run("refresh nil + still listed keeps the resource and errors", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{
			read: func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			listStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
				return []*client.PublicCloudVMInstance{{ID: "vm-1"}}, nil
			},
		}
		diags := readVMInstanceInto(context.Background(), d, funcs, vmInstanceReadForRefresh)
		if !diags.HasError() {
			t.Fatal("a VM still present in the strict listing must not be dropped")
		}
		if d.Id() != "vm-1" {
			t.Fatal("a still-listed VM must be kept")
		}
	})

	t.Run("refresh nil + listing fails keeps the resource and errors", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{
			read:       func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			listStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) { return nil, errors.New("206") },
		}
		diags := readVMInstanceInto(context.Background(), d, funcs, vmInstanceReadForRefresh)
		if !diags.HasError() {
			t.Fatal("a failed strict listing must not drop the resource")
		}
		if d.Id() != "vm-1" {
			t.Fatal("an unconfirmed absence must keep the resource")
		}
	})

	t.Run("refresh nil + confirmed absent drops the resource", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{
			read: func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			listStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
				return []*client.PublicCloudVMInstance{{ID: "other"}}, nil
			},
		}
		diags := readVMInstanceInto(context.Background(), d, funcs, vmInstanceReadForRefresh)
		if diags.HasError() {
			t.Fatalf("a confirmed absence must drop cleanly, got %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("a VM absent from the strict listing must be dropped")
		}
	})

	t.Run("a body with a mismatched id fails closed (never adopts another VM)", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{read: func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error) {
			return &client.PublicCloudVMInstance{ID: "someone-else", Name: "web"}, nil
		}}
		diags := readVMInstanceInto(context.Background(), d, funcs, vmInstanceReadForRefresh)
		if !diags.HasError() {
			t.Fatal("a read returning a different id must fail closed")
		}
		if d.Id() != "vm-1" {
			t.Fatal("a mismatched read must keep the original id")
		}
	})

	t.Run("a case-different UUID is accepted (UUIDs are case-insensitive)", func(t *testing.T) {
		d := createRD(t)
		d.SetId("ABCDEF00-0000-0000-0000-000000000000")
		funcs := vmInstanceCRUDFuncs{
			read: func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: "abcdef00-0000-0000-0000-000000000000", Name: "web", Status: "running"}, nil
			},
			listDisks: okListPrimaryDisk,
		}
		if diags := readVMInstanceInto(context.Background(), d, funcs, vmInstanceReadForRefresh); diags.HasError() {
			t.Fatalf("a case-only UUID difference must not be rejected, got %v", diags)
		}
	})

	t.Run("readAfterWrite does not overwrite power_state from a lagging status", func(t *testing.T) {
		d := createRD(t) // power_state = "on" in config
		d.SetId("vm-1")
		// status lags at "stopped" right after a write; power_state must stay the
		// declared "on" (refresh mode is what reconciles power_state from status).
		funcs := vmInstanceCRUDFuncs{
			read: func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: "vm-1", Status: "stopped", VCPU: 1, RAMGb: 2}, nil
			},
			listDisks: okListPrimaryDisk,
		}
		if diags := readVMInstanceInto(context.Background(), d, funcs, vmInstanceReadAfterWrite); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Get("power_state").(string) != "on" {
			t.Fatalf("readAfterWrite must not overwrite the declared power_state, got %q", d.Get("power_state"))
		}
		if d.Get("status").(string) != "stopped" {
			t.Fatalf("status (computed) must still be set, got %q", d.Get("status"))
		}
	})
}

// --- request building / mapping ---

func TestBuildCreateVMInstanceRequest(t *testing.T) {
	d := createRD(t)
	req, err := buildCreateVMInstanceRequest(d)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if req.Name != "web" || req.CPU != 2 || req.Memory != 4 || req.PowerState != "on" {
		t.Fatalf("scalar mapping wrong: %+v", req)
	}
	if len(req.NetworkInterfaces) != 1 || req.NetworkInterfaces[0].NetworkID != "55555555-5555-5555-5555-555555555555" {
		t.Fatalf("nic mapping wrong: %+v", req.NetworkInterfaces)
	}
	if req.CloudInit != nil {
		t.Fatalf("cloud_init must be nil when unset, got %+v", req.CloudInit)
	}
}

func TestExpandVMInstanceCloudInit(t *testing.T) {
	if ci := expandVMInstanceCloudInit(map[string]interface{}{}); ci != nil {
		t.Fatalf("empty cloud_init must be nil, got %+v", ci)
	}
	ci := expandVMInstanceCloudInit(map[string]interface{}{"cloud_config": "#cloud-config", "network_config": "version: 2"})
	if ci == nil || ci.CloudConfig != "#cloud-config" || ci.NetworkConfig != "version: 2" {
		t.Fatalf("cloud_init mapping wrong: %+v", ci)
	}
}

func TestPowerStateFromStatus(t *testing.T) {
	// Only a definitively "stopped" VM is off; anything else (running or a
	// transitional/unknown status) is on, so a resize is never allowed against a
	// VM that is not fully stopped.
	cases := map[string]string{"stopped": "off", "Stopped": "off", "running": "on", "Running": "on", "": "on", "starting": "on", "stopping": "on"}
	for status, want := range cases {
		if got := powerStateFromStatus(status); got != want {
			t.Fatalf("powerStateFromStatus(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestDeleteVMInstanceWith(t *testing.T) {
	t.Run("delete waits for the activity and succeeds", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{
			del: func(ctx context.Context, id string) (string, error) { return "act-del", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return vmiCompletedActivity("", "vm-1"), nil
			},
		}
		if diags := deleteVMInstanceWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
	})

	t.Run("a delete error fails", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{
			del: func(ctx context.Context, id string) (string, error) { return "", errors.New("boom") },
		}
		if diags := deleteVMInstanceWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a delete error must fail")
		}
	})
}

func TestSetVMInstanceOSDisk(t *testing.T) {
	t.Run("populates os_disk and os_disk_size_gb from the primary disk", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{
			listDisks: func(ctx context.Context, id string) ([]*client.PublicCloudVMDisk, error) {
				return []*client.PublicCloudVMDisk{
					{ID: "data-1", Position: 1, SizeGb: 10, IsPrimary: false},
					{ID: "sys", Position: 0, SizeGb: 38, StorageType: "st-1", IsPrimary: true},
				}, nil
			},
		}
		if diags := setVMInstanceOSDisk(context.Background(), d, funcs, "vm-1"); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Get("os_disk_size_gb").(int) != 38 {
			t.Fatalf("os_disk_size_gb = %d, want 38 (primary size)", d.Get("os_disk_size_gb"))
		}
		osd := d.Get("os_disk").([]interface{})
		if len(osd) != 1 {
			t.Fatalf("os_disk must have exactly the primary disk, got %d", len(osd))
		}
		m := osd[0].(map[string]interface{})
		if m["id"] != "sys" || m["is_primary"] != true || m["size_gb"].(int) != 38 {
			t.Fatalf("os_disk primary not mapped: %v", m)
		}
	})

	t.Run("listDisks error fails closed", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		funcs := vmInstanceCRUDFuncs{
			listDisks: func(ctx context.Context, id string) ([]*client.PublicCloudVMDisk, error) {
				return nil, errors.New("403")
			},
		}
		if diags := setVMInstanceOSDisk(context.Background(), d, funcs, "vm-1"); !diags.HasError() {
			t.Fatal("a disk-listing error must fail closed")
		}
	})

	t.Run("no primary disk clears any stale os_disk without error", func(t *testing.T) {
		d := createRD(t)
		d.SetId("vm-1")
		// Seed a stale os_disk to prove the no-primary path clears it.
		if err := d.Set("os_disk", []map[string]interface{}{{"id": "stale", "size_gb": 99, "is_primary": true, "position": 0}}); err != nil {
			t.Fatalf("seed: %v", err)
		}
		funcs := vmInstanceCRUDFuncs{
			listDisks: func(ctx context.Context, id string) ([]*client.PublicCloudVMDisk, error) {
				return []*client.PublicCloudVMDisk{{ID: "data", Position: 1, IsPrimary: false}}, nil
			},
		}
		if diags := setVMInstanceOSDisk(context.Background(), d, funcs, "vm-1"); diags.HasError() {
			t.Fatalf("no primary must not error, got %v", diags)
		}
		if len(d.Get("os_disk").([]interface{})) != 0 {
			t.Fatal("os_disk must be cleared when there is no primary (no stale value)")
		}
	})
}
