package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	diskTestVMID   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	diskTestDiskID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func newDiskRD(t *testing.T, size int) *schema.ResourceData {
	t.Helper()
	return schema.TestResourceDataRaw(t, resourcePublicCloudVMDisk().Schema, map[string]interface{}{
		"virtual_machine_id": diskTestVMID,
		"size":               size,
	})
}

func dataDisk(id string, size int) *client.PublicCloudVMDisk {
	return &client.PublicCloudVMDisk{ID: id, Position: 1, Label: "data", SizeGb: size, StorageType: "st-1", IsPrimary: false}
}

func diskActivity(result string) *client.Activity {
	return &client.Activity{State: map[string]client.ActivityState{"completed": {Result: result}}}
}

func TestVMDiskGrowOnlyCheck(t *testing.T) {
	cases := []struct {
		old, new int
		wantErr  bool
	}{
		{0, 10, false},  // create
		{10, 20, false}, // grow
		{10, 10, false}, // no change
		{20, 10, true},  // shrink
	}
	for _, tc := range cases {
		if err := vmDiskGrowOnlyCheck(tc.old, tc.new); (err != nil) != tc.wantErr {
			t.Fatalf("vmDiskGrowOnlyCheck(%d,%d) err=%v wantErr=%v", tc.old, tc.new, err, tc.wantErr)
		}
	}
}

func TestSingleActivityResult(t *testing.T) {
	if singleActivityResult(nil) != "" {
		t.Fatal("nil activity must yield empty")
	}
	if singleActivityResult(diskActivity("x")) != "x" {
		t.Fatal("single-state result must be returned")
	}
	two := &client.Activity{State: map[string]client.ActivityState{"a": {Result: "x"}, "b": {Result: "y"}}}
	if singleActivityResult(two) != "" {
		t.Fatal("multi-state activity must yield empty (ambiguous)")
	}
}

func TestCreateVMDiskWith(t *testing.T) {
	okRead := func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
		return dataDisk(diskID, 10), nil
	}

	t.Run("id from validated activity result, non-primary readback", func(t *testing.T) {
		d := newDiskRD(t, 10)
		funcs := vmDiskCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMDiskRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return diskActivity(diskTestDiskID), nil
			},
			read: okRead,
		}
		if diags := createVMDiskWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Id() != diskTestDiskID {
			t.Fatalf("id = %q, want %q", d.Id(), diskTestDiskID)
		}
	})

	t.Run("result == vmID fails closed without SetId", func(t *testing.T) {
		d := newDiskRD(t, 10)
		funcs := vmDiskCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMDiskRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) { return diskActivity(diskTestVMID), nil },
		}
		if diags := createVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a result equal to the vmID must fail closed")
		}
		if d.Id() != "" {
			t.Fatalf("no id must be set, got %q", d.Id())
		}
	})

	t.Run("empty and non-UUID results fail closed", func(t *testing.T) {
		for _, res := range []string{"", "not-a-uuid"} {
			d := newDiskRD(t, 10)
			funcs := vmDiskCRUDFuncs{
				create: func(ctx context.Context, vmID string, r *client.CreateVMDiskRequest) (string, error) {
					return "act", nil
				},
				waitActivity: func(ctx context.Context, a string) (*client.Activity, error) { return diskActivity(res), nil },
			}
			if diags := createVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
				t.Fatalf("result %q must fail closed", res)
			}
			if d.Id() != "" {
				t.Fatalf("result %q must not set an id", res)
			}
		}
	})

	t.Run("wait failure fails without SetId", func(t *testing.T) {
		d := newDiskRD(t, 10)
		funcs := vmDiskCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMDiskRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return nil, errors.New("activity failed")
			},
		}
		if diags := createVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a wait failure must fail the create")
		}
		if d.Id() != "" {
			t.Fatal("a wait failure must not set an id")
		}
	})

	t.Run("readback of a primary disk fails", func(t *testing.T) {
		d := newDiskRD(t, 10)
		funcs := vmDiskCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMDiskRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return diskActivity(diskTestDiskID), nil
			},
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				p := dataDisk(diskID, 10)
				p.IsPrimary = true
				p.Position = 0
				return p, nil
			},
		}
		if diags := createVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a primary disk read-back must fail (data-disk resource only)")
		}
		if d.Id() != "" {
			t.Fatal("a primary read-back is a proven wrong-adoption and must clear the id (never leave the system disk id in state)")
		}
	})

	t.Run("unreadable read-back (error) keeps the fresh id", func(t *testing.T) {
		d := newDiskRD(t, 10)
		funcs := vmDiskCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMDiskRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return diskActivity(diskTestDiskID), nil
			},
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				return nil, errors.New("503")
			},
		}
		if diags := createVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("an unreadable read-back must fail the create")
		}
		if d.Id() != diskTestDiskID {
			t.Fatalf("an unreadable read-back is eventual consistency and must keep the fresh id, got %q", d.Id())
		}
	})

	t.Run("not-yet-readable read-back (nil) keeps the fresh id", func(t *testing.T) {
		d := newDiskRD(t, 10)
		funcs := vmDiskCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMDiskRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return diskActivity(diskTestDiskID), nil
			},
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) { return nil, nil },
		}
		if diags := createVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a not-yet-readable read-back must fail the create")
		}
		if d.Id() != diskTestDiskID {
			t.Fatalf("a nil read-back is eventual consistency and must keep the fresh id, got %q", d.Id())
		}
	})

	t.Run("size mismatch on readback fails", func(t *testing.T) {
		d := newDiskRD(t, 10)
		funcs := vmDiskCRUDFuncs{
			create: func(ctx context.Context, vmID string, r *client.CreateVMDiskRequest) (string, error) {
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return diskActivity(diskTestDiskID), nil
			},
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				return dataDisk(diskID, 999), nil
			},
		}
		if diags := createVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a size mismatch on read-back must fail (inconsistent id)")
		}
		if d.Id() != "" {
			t.Fatal("a proven wrong-adoption must clear the id (never leave a wrong disk in state to be destroyed later)")
		}
	})
}

func TestReadVMDiskInto(t *testing.T) {
	setup := func(t *testing.T) *schema.ResourceData {
		d := newDiskRD(t, 10)
		d.SetId(diskTestDiskID)
		return d
	}

	t.Run("data disk found sets state", func(t *testing.T) {
		d := setup(t)
		funcs := vmDiskCRUDFuncs{read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
			return dataDisk(diskID, 12), nil
		}}
		if diags := readVMDiskInto(context.Background(), d, funcs, vmDiskReadForRefresh); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Get("size").(int) != 12 || d.Get("is_primary").(bool) {
			t.Fatalf("state not set: size=%d primary=%v", d.Get("size"), d.Get("is_primary"))
		}
	})

	t.Run("primary disk found is refused (kept)", func(t *testing.T) {
		d := setup(t)
		funcs := vmDiskCRUDFuncs{read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
			p := dataDisk(diskID, 10)
			p.IsPrimary = true
			return p, nil
		}}
		if diags := readVMDiskInto(context.Background(), d, funcs, vmDiskReadForRefresh); !diags.HasError() {
			t.Fatal("a primary disk must be refused")
		}
		if d.Id() == "" {
			t.Fatal("a refused primary must keep the resource")
		}
	})

	t.Run("read error keeps the resource", func(t *testing.T) {
		d := setup(t)
		funcs := vmDiskCRUDFuncs{read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
			return nil, errors.New("403")
		}}
		if diags := readVMDiskInto(context.Background(), d, funcs, vmDiskReadForRefresh); !diags.HasError() {
			t.Fatal("a read error must fail closed")
		}
		if d.Id() == "" {
			t.Fatal("a read error must not drop")
		}
	})

	t.Run("afterWrite nil keeps id", func(t *testing.T) {
		d := setup(t)
		funcs := vmDiskCRUDFuncs{read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) { return nil, nil }}
		if diags := readVMDiskInto(context.Background(), d, funcs, vmDiskReadAfterWrite); !diags.HasError() {
			t.Fatal("afterWrite nil must fail closed")
		}
		if d.Id() == "" {
			t.Fatal("afterWrite must never drop the fresh id")
		}
	})

	nilDisk := func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) { return nil, nil }

	t.Run("refresh nil + VM gone (not listed) drops", func(t *testing.T) {
		d := setup(t)
		funcs := vmDiskCRUDFuncs{
			read:   nilDisk,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
				return []*client.PublicCloudVMInstance{{ID: "other"}}, nil
			},
		}
		if diags := readVMDiskInto(context.Background(), d, funcs, vmDiskReadForRefresh); diags.HasError() {
			t.Fatalf("a confirmed-gone VM must drop the disk cleanly, got %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("disk of a gone VM must be dropped")
		}
	})

	t.Run("refresh nil + VM gone but still listed keeps", func(t *testing.T) {
		d := setup(t)
		funcs := vmDiskCRUDFuncs{
			read:   nilDisk,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
				return []*client.PublicCloudVMInstance{{ID: diskTestVMID}}, nil
			},
		}
		if diags := readVMDiskInto(context.Background(), d, funcs, vmDiskReadForRefresh); !diags.HasError() {
			t.Fatal("a VM still listed must not drop the disk")
		}
		if d.Id() == "" {
			t.Fatal("must keep")
		}
	})

	t.Run("refresh nil + VM listing fails keeps", func(t *testing.T) {
		d := setup(t)
		funcs := vmDiskCRUDFuncs{
			read:         nilDisk,
			vmRead:       func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) { return nil, errors.New("206") },
		}
		if diags := readVMDiskInto(context.Background(), d, funcs, vmDiskReadForRefresh); !diags.HasError() {
			t.Fatal("a failed VM listing must not drop")
		}
	})

	t.Run("refresh nil + VM exists + disk absent drops", func(t *testing.T) {
		d := setup(t)
		funcs := vmDiskCRUDFuncs{
			read: nilDisk,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMDisk, error) {
				return []*client.PublicCloudVMDisk{{ID: "other"}}, nil
			},
		}
		if diags := readVMDiskInto(context.Background(), d, funcs, vmDiskReadForRefresh); diags.HasError() {
			t.Fatalf("a disk absent from a complete listing must drop, got %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("absent disk must be dropped")
		}
	})

	t.Run("refresh nil + VM exists + disk still listed keeps", func(t *testing.T) {
		d := setup(t)
		funcs := vmDiskCRUDFuncs{
			read: nilDisk,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMDisk, error) {
				return []*client.PublicCloudVMDisk{{ID: diskTestDiskID}}, nil
			},
		}
		if diags := readVMDiskInto(context.Background(), d, funcs, vmDiskReadForRefresh); !diags.HasError() {
			t.Fatal("a still-listed disk must not drop")
		}
		if d.Id() == "" {
			t.Fatal("must keep")
		}
	})

	t.Run("refresh nil + VM exists + disk listing fails keeps", func(t *testing.T) {
		d := setup(t)
		funcs := vmDiskCRUDFuncs{
			read: nilDisk,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMDisk, error) {
				return nil, errors.New("total mismatch")
			},
		}
		if diags := readVMDiskInto(context.Background(), d, funcs, vmDiskReadForRefresh); !diags.HasError() {
			t.Fatal("a failed disk listing must not drop")
		}
	})
}

func TestUpdateVMDiskWith(t *testing.T) {
	// TestResourceDataRaw cannot model a diff, so exercise the extend logic via a
	// resource data whose size is set (HasChange is true vs zero) — the preflight
	// paths are what matter for state safety.
	newSized := func(t *testing.T) *schema.ResourceData {
		d := newDiskRD(t, 20)
		d.SetId(diskTestDiskID)
		return d
	}

	t.Run("extend refused while VM running (no extend call)", func(t *testing.T) {
		d := newSized(t)
		extended := false
		funcs := vmDiskCRUDFuncs{
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				return dataDisk(diskID, 10), nil
			},
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "running"}, nil
			},
			extend: func(ctx context.Context, vmID, diskID string, size int) (string, error) {
				extended = true
				return "act", nil
			},
		}
		if diags := updateVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("extending while the VM is running must fail")
		}
		if extended {
			t.Fatal("extend must NOT be called when the VM is running")
		}
	})

	t.Run("extend refused on a primary disk", func(t *testing.T) {
		d := newSized(t)
		funcs := vmDiskCRUDFuncs{
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				p := dataDisk(diskID, 10)
				p.IsPrimary = true
				return p, nil
			},
		}
		if diags := updateVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("extending a primary disk must fail")
		}
	})

	t.Run("extend proceeds when VM stopped", func(t *testing.T) {
		d := newSized(t)
		extended := false
		funcs := vmDiskCRUDFuncs{
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				return dataDisk(diskID, 20), nil
			},
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			extend: func(ctx context.Context, vmID, diskID string, size int) (string, error) {
				extended = true
				return "act", nil
			},
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return diskActivity(diskTestDiskID), nil
			},
		}
		if diags := updateVMDiskWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("extend on a stopped VM must succeed, got %v", diags)
		}
		if !extended {
			t.Fatal("extend must be called when the VM is stopped")
		}
	})
}

func TestDeleteVMDiskWith(t *testing.T) {
	t.Run("data disk deleted when VM stopped", func(t *testing.T) {
		d := newDiskRD(t, 10)
		d.SetId(diskTestDiskID)
		deleted := false
		funcs := vmDiskCRUDFuncs{
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				return dataDisk(diskID, 10), nil
			},
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			del: func(ctx context.Context, vmID, diskID string) (string, error) { deleted = true; return "act", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return diskActivity(diskTestDiskID), nil
			},
		}
		if diags := deleteVMDiskWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("delete: %v", diags)
		}
		if !deleted {
			t.Fatal("delete must call the API")
		}
	})

	t.Run("delete refused while VM running (no del call)", func(t *testing.T) {
		d := newDiskRD(t, 10)
		d.SetId(diskTestDiskID)
		deleted := false
		funcs := vmDiskCRUDFuncs{
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				return dataDisk(diskID, 10), nil
			},
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "running"}, nil
			},
			del: func(ctx context.Context, vmID, diskID string) (string, error) { deleted = true; return "act", nil },
		}
		if diags := deleteVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("deleting a data disk while the VM is running must be refused")
		}
		if deleted {
			t.Fatal("delete must NOT be called while the VM is running")
		}
	})

	t.Run("primary disk delete refused (no del call)", func(t *testing.T) {
		d := newDiskRD(t, 10)
		d.SetId(diskTestDiskID)
		deleted := false
		funcs := vmDiskCRUDFuncs{
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				p := dataDisk(diskID, 10)
				p.IsPrimary = true
				return p, nil
			},
			del: func(ctx context.Context, vmID, diskID string) (string, error) { deleted = true; return "act", nil },
		}
		if diags := deleteVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("deleting a primary disk must be refused")
		}
		if deleted {
			t.Fatal("delete must NOT be called for a primary disk")
		}
	})

	t.Run("delete refused when disk present but VM read is nil (ambiguous)", func(t *testing.T) {
		d := newDiskRD(t, 10)
		d.SetId(diskTestDiskID)
		deleted := false
		funcs := vmDiskCRUDFuncs{
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				return dataDisk(diskID, 10), nil
			},
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			del:    func(ctx context.Context, vmID, diskID string) (string, error) { deleted = true; return "act", nil },
		}
		if diags := deleteVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a present disk with a nil VM read is ambiguous and must not be deleted")
		}
		if deleted {
			t.Fatal("delete must NOT be called on an ambiguous signal")
		}
	})

	t.Run("delete 404 race confirms absence before accepting", func(t *testing.T) {
		d := newDiskRD(t, 10)
		d.SetId(diskTestDiskID)
		funcs := vmDiskCRUDFuncs{
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				return dataDisk(diskID, 10), nil
			},
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			del: func(ctx context.Context, vmID, diskID string) (string, error) {
				return "", client.StatusError{Code: 404}
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMDisk, error) {
				return []*client.PublicCloudVMDisk{}, nil
			},
		}
		if diags := deleteVMDiskWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("a 404-on-delete with confirmed absence must succeed, got %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("a confirmed-absent disk must be dropped")
		}
	})

	t.Run("delete 404 race but disk still listed keeps (never drop on a bare 404)", func(t *testing.T) {
		d := newDiskRD(t, 10)
		d.SetId(diskTestDiskID)
		funcs := vmDiskCRUDFuncs{
			read: func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) {
				return dataDisk(diskID, 10), nil
			},
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID, Status: "stopped"}, nil
			},
			del: func(ctx context.Context, vmID, diskID string) (string, error) {
				return "", client.StatusError{Code: 404}
			},
			// The strict listing still shows the disk: a 404 must NOT be trusted as
			// deletion evidence — keep the resource and error.
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMDisk, error) {
				return []*client.PublicCloudVMDisk{dataDisk(diskTestDiskID, 10)}, nil
			},
		}
		if diags := deleteVMDiskWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a 404 while the disk is still listed must keep the resource and error (never a bare-404 drop)")
		}
		if d.Id() == "" {
			t.Fatal("must not drop on an unconfirmed 404")
		}
	})

	t.Run("already-absent disk confirmed gone via VM absence", func(t *testing.T) {
		d := newDiskRD(t, 10)
		d.SetId(diskTestDiskID)
		funcs := vmDiskCRUDFuncs{
			read:   func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error) { return nil, nil },
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
				return []*client.PublicCloudVMInstance{}, nil
			},
		}
		if diags := deleteVMDiskWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("an already-absent disk (VM gone) must delete cleanly, got %v", diags)
		}
	})
}
