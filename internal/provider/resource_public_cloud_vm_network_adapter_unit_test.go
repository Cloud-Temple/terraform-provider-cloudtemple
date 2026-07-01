package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	nicTestVMID       = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	nicTestNICID      = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	nicTestNetworkID  = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	nicTestNetworkID2 = "dddddddd-dddd-dddd-dddd-dddddddddddd"
)

func newNICRD(t *testing.T, deviceIndex int, networkID, ipAddress string) *schema.ResourceData {
	t.Helper()
	raw := map[string]interface{}{
		"virtual_machine_id": nicTestVMID,
		"device_index":       deviceIndex,
		"network_id":         networkID,
	}
	if ipAddress != "" {
		raw["ip_address"] = ipAddress
	}
	return schema.TestResourceDataRaw(t, resourcePublicCloudVMNetworkAdapter().Schema, raw)
}

func nicAdapter(id string, deviceIndex int, networkID, typ string) *client.PublicCloudVMNetworkAdapter {
	return &client.PublicCloudVMNetworkAdapter{
		ID:              id,
		DeviceIndex:     deviceIndex,
		NetworkID:       networkID,
		NetworkName:     "net",
		Type:            typ,
		ProvisionStatus: "provisioned",
	}
}

// nicActivityConcerned models the create activity: a network_adapter concerned
// item carries the new NIC id (plus a single completed state).
func nicActivityConcerned(nicID string) *client.Activity {
	return &client.Activity{
		ConcernedItems: []client.ActivityConcernedItem{
			{Type: "vmi", ID: nicTestVMID},
			{Type: "network_adapter", ID: nicID},
		},
		State: map[string]client.ActivityState{"completed": {Result: nicID}},
	}
}

// nicActivityResultOnly models an activity with NO network_adapter concerned item
// (only a single result) — the fallback id source.
func nicActivityResultOnly(result string) *client.Activity {
	return &client.Activity{State: map[string]client.ActivityState{"completed": {Result: result}}}
}

func TestCreateVMNICWith(t *testing.T) {
	okRead := func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
		return nicAdapter(nicID, 1, nicTestNetworkID, "private_backbone"), nil
	}

	t.Run("id from network_adapter concerned item, identity confirmed", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		funcs := vmNICCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMNetworkAdapterRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nicActivityConcerned(nicTestNICID), nil
			},
			read: okRead,
		}
		if diags := createVMNICWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Id() != nicTestNICID {
			t.Fatalf("id = %q, want %q", d.Id(), nicTestNICID)
		}
	})

	t.Run("id falls back to single result when no concerned item", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		funcs := vmNICCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMNetworkAdapterRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nicActivityResultOnly(nicTestNICID), nil
			},
			read: okRead,
		}
		if diags := createVMNICWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Id() != nicTestNICID {
			t.Fatalf("id = %q, want %q", d.Id(), nicTestNICID)
		}
	})

	t.Run("result == vmID fails closed without SetId", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		funcs := vmNICCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMNetworkAdapterRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nicActivityResultOnly(nicTestVMID), nil
			},
		}
		if diags := createVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a result equal to the vmID must fail closed")
		}
		if d.Id() != "" {
			t.Fatalf("no id must be set, got %q", d.Id())
		}
	})

	t.Run("empty and non-UUID ids fail closed", func(t *testing.T) {
		for _, res := range []string{"", "not-a-uuid"} {
			d := newNICRD(t, 1, nicTestNetworkID, "")
			funcs := vmNICCRUDFuncs{
				create: func(ctx context.Context, vmID string, r *client.CreateVMNetworkAdapterRequest) (string, error) {
					return "act", nil
				},
				waitActivity: func(ctx context.Context, a string) (*client.Activity, error) { return nicActivityResultOnly(res), nil },
			}
			if diags := createVMNICWith(context.Background(), d, funcs); !diags.HasError() {
				t.Fatalf("id %q must fail closed", res)
			}
			if d.Id() != "" {
				t.Fatalf("id %q must not set a resource id", res)
			}
		}
	})

	t.Run("wait failure fails without SetId", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		funcs := vmNICCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMNetworkAdapterRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nil, errors.New("activity failed")
			},
		}
		if diags := createVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a wait failure must fail the create")
		}
		if d.Id() != "" {
			t.Fatal("a wait failure must not set an id")
		}
	})

	t.Run("device_index mismatch on read-back fails (wrong nic adopted)", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		funcs := vmNICCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMNetworkAdapterRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nicActivityConcerned(nicTestNICID), nil
			},
			read: func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
				return nicAdapter(nicID, 99, nicTestNetworkID, "private_backbone"), nil
			},
		}
		if diags := createVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a device_index mismatch on read-back must fail closed")
		}
		if d.Id() != "" {
			t.Fatal("a proven wrong-adoption must clear the id (never leave a wrong nic in state to be destroyed later)")
		}
	})

	t.Run("network_id mismatch on read-back fails", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		funcs := vmNICCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMNetworkAdapterRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nicActivityConcerned(nicTestNICID), nil
			},
			read: func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
				return nicAdapter(nicID, 1, nicTestNetworkID2, "private_backbone"), nil
			},
		}
		if diags := createVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a network_id mismatch on read-back must fail closed")
		}
		if d.Id() != "" {
			t.Fatal("a proven wrong-adoption must clear the id (never leave a wrong nic in state to be destroyed later)")
		}
	})
}

func TestReadVMNICInto(t *testing.T) {
	setup := func(t *testing.T, ip string) *schema.ResourceData {
		d := newNICRD(t, 1, nicTestNetworkID, ip)
		d.SetId(nicTestNICID)
		return d
	}
	nilNIC := func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
		return nil, nil
	}
	errNIC := func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
		return nil, client.StatusError{Code: 400}
	}

	t.Run("found sets state", func(t *testing.T) {
		d := setup(t, "")
		funcs := vmNICCRUDFuncs{read: func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
			return nicAdapter(nicID, 1, nicTestNetworkID, "vpc"), nil
		}}
		if diags := readVMNICInto(context.Background(), d, funcs, vmNICReadForRefresh); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Get("type").(string) != "vpc" || d.Get("network_id").(string) != nicTestNetworkID {
			t.Fatalf("state not set: %v / %v", d.Get("type"), d.Get("network_id"))
		}
	})

	t.Run("ip_address on a non-VPC network warns (no error, no perpetual diff)", func(t *testing.T) {
		d := setup(t, "10.0.0.5")
		funcs := vmNICCRUDFuncs{read: func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
			// non-VPC network, and the platform reports no ipv4 for the static ip.
			return nicAdapter(nicID, 1, nicTestNetworkID, "private_backbone"), nil
		}}
		diags := readVMNICInto(context.Background(), d, funcs, vmNICReadForRefresh)
		if diags.HasError() {
			t.Fatalf("a non-VPC ip_address must warn, not error: %v", diags)
		}
		hasWarning := false
		for _, dg := range diags {
			if dg.Severity == diag.Warning {
				hasWarning = true
			}
		}
		if !hasWarning {
			t.Fatal("expected a warning that ip_address has no effect on a non-VPC network")
		}
		// ip_address must remain the configured value (never overwritten from ipv4_address).
		if d.Get("ip_address").(string) != "10.0.0.5" {
			t.Fatalf("ip_address must keep the configured value, got %q", d.Get("ip_address"))
		}
	})

	t.Run("read error keeps the resource (refresh, ambiguous)", func(t *testing.T) {
		d := setup(t, "")
		// A 400 read error with a VM that is up and still lists the nic must keep.
		funcs := vmNICCRUDFuncs{
			read: errNIC,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMNetworkAdapter, error) {
				return []*client.PublicCloudVMNetworkAdapter{nicAdapter(nicTestNICID, 1, nicTestNetworkID, "vpc")}, nil
			},
		}
		if diags := readVMNICInto(context.Background(), d, funcs, vmNICReadForRefresh); !diags.HasError() {
			t.Fatal("a 400 read while the nic is still listed must keep + error")
		}
		if d.Id() == "" {
			t.Fatal("must not drop on an ambiguous read")
		}
	})

	t.Run("refresh 400 + complete listing missing nic drops", func(t *testing.T) {
		d := setup(t, "")
		funcs := vmNICCRUDFuncs{
			read: errNIC,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMNetworkAdapter, error) {
				return []*client.PublicCloudVMNetworkAdapter{nicAdapter("other", 2, nicTestNetworkID, "vpc")}, nil
			},
		}
		if diags := readVMNICInto(context.Background(), d, funcs, vmNICReadForRefresh); diags.HasError() {
			t.Fatalf("a nic absent from a complete listing must drop, got %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("absent nic must be dropped")
		}
	})

	t.Run("afterWrite error keeps id", func(t *testing.T) {
		d := setup(t, "")
		funcs := vmNICCRUDFuncs{read: errNIC}
		if diags := readVMNICInto(context.Background(), d, funcs, vmNICReadAfterWrite); !diags.HasError() {
			t.Fatal("afterWrite read error must fail closed")
		}
		if d.Id() == "" {
			t.Fatal("afterWrite must never drop the fresh id")
		}
	})

	t.Run("afterWrite nil keeps id", func(t *testing.T) {
		d := setup(t, "")
		funcs := vmNICCRUDFuncs{read: nilNIC}
		if diags := readVMNICInto(context.Background(), d, funcs, vmNICReadAfterWrite); !diags.HasError() {
			t.Fatal("afterWrite nil must fail closed")
		}
		if d.Id() == "" {
			t.Fatal("afterWrite must never drop the fresh id even when the complete listing would omit it")
		}
	})

	t.Run("refresh nil + VM gone (not listed) drops", func(t *testing.T) {
		d := setup(t, "")
		funcs := vmNICCRUDFuncs{
			read:   nilNIC,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
				return []*client.PublicCloudVMInstance{{ID: "other"}}, nil
			},
		}
		if diags := readVMNICInto(context.Background(), d, funcs, vmNICReadForRefresh); diags.HasError() {
			t.Fatalf("a confirmed-gone VM must drop the nic cleanly, got %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("nic of a gone VM must be dropped")
		}
	})

	t.Run("refresh nil + VM gone but still listed keeps", func(t *testing.T) {
		d := setup(t, "")
		funcs := vmNICCRUDFuncs{
			read:   nilNIC,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
				return []*client.PublicCloudVMInstance{{ID: nicTestVMID}}, nil
			},
		}
		if diags := readVMNICInto(context.Background(), d, funcs, vmNICReadForRefresh); !diags.HasError() {
			t.Fatal("a VM still listed must not drop the nic")
		}
		if d.Id() == "" {
			t.Fatal("must keep")
		}
	})

	t.Run("refresh nil + VM listing fails keeps", func(t *testing.T) {
		d := setup(t, "")
		funcs := vmNICCRUDFuncs{
			read:         nilNIC,
			vmRead:       func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) { return nil, errors.New("206") },
		}
		if diags := readVMNICInto(context.Background(), d, funcs, vmNICReadForRefresh); !diags.HasError() {
			t.Fatal("a failed VM listing must not drop")
		}
	})

	t.Run("refresh nil + VM exists + nic listing fails keeps", func(t *testing.T) {
		d := setup(t, "")
		funcs := vmNICCRUDFuncs{
			read: nilNIC,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMNetworkAdapter, error) {
				return nil, errors.New("total mismatch")
			},
		}
		if diags := readVMNICInto(context.Background(), d, funcs, vmNICReadForRefresh); !diags.HasError() {
			t.Fatal("a failed nic listing must not drop")
		}
	})
}

func TestUpdateVMNICWith(t *testing.T) {
	// TestResourceDataRaw cannot model a diff; a set network_id reads as a change
	// vs the zero value, which drives the change-network path we want to exercise.
	newChanged := func(t *testing.T) *schema.ResourceData {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		d.SetId(nicTestNICID)
		return d
	}

	t.Run("change refused while VM running (no changeNetwork call)", func(t *testing.T) {
		d := newChanged(t)
		called := false
		funcs := vmNICCRUDFuncs{
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "running"}, nil
			},
			changeNetwork: func(ctx context.Context, vmID, nicID string, r *client.ChangeVMNetworkAdapterRequest) (string, error) {
				called = true
				return "act", nil
			},
		}
		if diags := updateVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("changing the network while the VM is running must fail")
		}
		if called {
			t.Fatal("changeNetwork must NOT be called when the VM is running")
		}
	})

	t.Run("change proceeds when VM stopped and read-back confirms", func(t *testing.T) {
		d := newChanged(t)
		called := false
		funcs := vmNICCRUDFuncs{
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			changeNetwork: func(ctx context.Context, vmID, nicID string, r *client.ChangeVMNetworkAdapterRequest) (string, error) {
				called = true
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nicActivityResultOnly(nicTestVMID), nil
			},
			read: func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
				return nicAdapter(nicID, 1, nicTestNetworkID, "vpc"), nil
			},
		}
		if diags := updateVMNICWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("change on a stopped VM must succeed, got %v", diags)
		}
		if !called {
			t.Fatal("changeNetwork must be called when the VM is stopped")
		}
		// The change activity's result is the vmId; it must NEVER be adopted as the
		// NIC id — the NIC id is unchanged.
		if d.Id() != nicTestNICID {
			t.Fatalf("update must keep the NIC id %q (activity result vmId must not be re-derived), got %q", nicTestNICID, d.Id())
		}
	})

	t.Run("change wait failure keeps id and fails", func(t *testing.T) {
		d := newChanged(t)
		funcs := vmNICCRUDFuncs{
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			changeNetwork: func(ctx context.Context, vmID, nicID string, r *client.ChangeVMNetworkAdapterRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nil, errors.New("activity failed")
			},
		}
		if diags := updateVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a change-network wait failure must fail")
		}
		if d.Id() != nicTestNICID {
			t.Fatal("a failed change must keep the resource id in state")
		}
	})

	t.Run("change activity completes but network did not change (hot-plug no-op) fails closed", func(t *testing.T) {
		d := newChanged(t)
		funcs := vmNICCRUDFuncs{
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			changeNetwork: func(ctx context.Context, vmID, nicID string, r *client.ChangeVMNetworkAdapterRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nicActivityResultOnly(nicTestVMID), nil
			},
			read: func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
				// read-back still on the OLD network — the hot-plug did not take effect.
				return nicAdapter(nicID, 1, nicTestNetworkID2, "vpc"), nil
			},
		}
		if diags := updateVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a change whose read-back network_id does not match the request must fail closed")
		}
		if d.Id() == "" {
			t.Fatal("a failed change must keep the resource in state")
		}
	})
}

func TestDeleteVMNICWith(t *testing.T) {
	present := func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
		return nicAdapter(nicID, 1, nicTestNetworkID, "vpc"), nil
	}

	t.Run("deleted when VM stopped", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		d.SetId(nicTestNICID)
		deleted := false
		funcs := vmNICCRUDFuncs{
			read: present,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			del: func(ctx context.Context, vmID, nicID string) (string, error) { deleted = true; return "act", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nicActivityResultOnly(nicTestVMID), nil
			},
		}
		if diags := deleteVMNICWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("delete: %v", diags)
		}
		if !deleted {
			t.Fatal("delete must call the API")
		}
	})

	t.Run("delete refused while VM running (no del call)", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		d.SetId(nicTestNICID)
		deleted := false
		funcs := vmNICCRUDFuncs{
			read: present,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "running"}, nil
			},
			del: func(ctx context.Context, vmID, nicID string) (string, error) { deleted = true; return "act", nil },
		}
		if diags := deleteVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("deleting while the VM is running must be refused")
		}
		if deleted {
			t.Fatal("delete must NOT be called while the VM is running")
		}
	})

	t.Run("delete refused when nic present but VM read is nil (ambiguous)", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		d.SetId(nicTestNICID)
		deleted := false
		funcs := vmNICCRUDFuncs{
			read:   present,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			del:    func(ctx context.Context, vmID, nicID string) (string, error) { deleted = true; return "act", nil },
		}
		if diags := deleteVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a present nic with a nil VM read is ambiguous and must not be deleted")
		}
		if deleted {
			t.Fatal("delete must NOT be called on an ambiguous signal")
		}
	})

	t.Run("delete 404 race confirms absence before accepting", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		d.SetId(nicTestNICID)
		funcs := vmNICCRUDFuncs{
			read: present,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			del: func(ctx context.Context, vmID, nicID string) (string, error) {
				return "", client.StatusError{Code: 404}
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMNetworkAdapter, error) {
				return []*client.PublicCloudVMNetworkAdapter{}, nil
			},
		}
		if diags := deleteVMNICWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("a 404-on-delete with confirmed absence must succeed, got %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("a confirmed-absent nic must be dropped")
		}
	})

	t.Run("delete 404 race but nic still listed keeps (never drop on a bare 404)", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		d.SetId(nicTestNICID)
		funcs := vmNICCRUDFuncs{
			read: present,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			del: func(ctx context.Context, vmID, nicID string) (string, error) {
				return "", client.StatusError{Code: 404}
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMNetworkAdapter, error) {
				return []*client.PublicCloudVMNetworkAdapter{nicAdapter(nicTestNICID, 1, nicTestNetworkID, "vpc")}, nil
			},
		}
		if diags := deleteVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a 404 while the nic is still listed must keep + error (never a bare-404 drop)")
		}
		if d.Id() == "" {
			t.Fatal("must not drop on an unconfirmed 404")
		}
	})

	t.Run("delete wait failure keeps the resource in state", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		d.SetId(nicTestNICID)
		funcs := vmNICCRUDFuncs{
			read: present,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			del: func(ctx context.Context, vmID, nicID string) (string, error) { return "act", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nil, errors.New("activity failed")
			},
		}
		if diags := deleteVMNICWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a delete wait failure must fail")
		}
		if d.Id() == "" {
			t.Fatal("a failed delete must keep the resource in state (never drop on an incomplete activity)")
		}
	})

	t.Run("already-absent nic confirmed gone via VM absence", func(t *testing.T) {
		d := newNICRD(t, 1, nicTestNetworkID, "")
		d.SetId(nicTestNICID)
		funcs := vmNICCRUDFuncs{
			read: func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error) {
				return nil, nil
			},
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
				return []*client.PublicCloudVMInstance{}, nil
			},
		}
		if diags := deleteVMNICWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("an already-absent nic (VM gone) must delete cleanly, got %v", diags)
		}
	})
}
