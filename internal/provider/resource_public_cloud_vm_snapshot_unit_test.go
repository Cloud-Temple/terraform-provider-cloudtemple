package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	snapTestVMID   = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	snapTestSnapID = "dddddddd-dddd-dddd-dddd-dddddddddddd"
)

func newSnapRD(t *testing.T) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourcePublicCloudVMSnapshot().Schema, map[string]interface{}{
		"virtual_machine_id": snapTestVMID,
		"name":               "snap",
	})
	return d
}

func snapActivity(concernedID, result string) *client.Activity {
	a := &client.Activity{State: map[string]client.ActivityState{"completed": {Result: result}}}
	if concernedID != "" {
		a.ConcernedItems = []client.ActivityConcernedItem{{ID: snapTestVMID, Type: "vmi"}, {ID: concernedID, Type: "snapshot"}}
	}
	return a
}

func snapObj(id string) *client.PublicCloudVMSnapshot {
	return &client.PublicCloudVMSnapshot{ID: id, VmID: snapTestVMID, Name: "snap", Status: "available", CreatedAt: "2026-07-01T12:00:00"}
}

func TestActivityConcernedItemID(t *testing.T) {
	if activityConcernedItemID(nil, "snapshot") != "" {
		t.Fatal("nil activity must yield empty")
	}
	a := snapActivity(snapTestSnapID, snapTestSnapID)
	if activityConcernedItemID(a, "snapshot") != snapTestSnapID {
		t.Fatal("must return the snapshot concerned item id")
	}
	if activityConcernedItemID(a, "disk") != "" {
		t.Fatal("no matching type must yield empty")
	}
}

func TestCreateVMSnapshotWith(t *testing.T) {
	okRead := func(ctx context.Context, vmID, snapID string) (*client.PublicCloudVMSnapshot, error) {
		return snapObj(snapID), nil
	}

	t.Run("id from concernedItems snapshot", func(t *testing.T) {
		d := newSnapRD(t)
		funcs := vmSnapshotCRUDFuncs{
			create: func(ctx context.Context, vmID, name string) (string, error) { return "act", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return snapActivity(snapTestSnapID, snapTestVMID), nil
			},
			read: okRead,
		}
		if diags := createVMSnapshotWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Id() != snapTestSnapID {
			t.Fatalf("id = %q, want %q", d.Id(), snapTestSnapID)
		}
		if d.Get("status").(string) != "available" {
			t.Fatalf("state not set: status=%q", d.Get("status"))
		}
	})

	t.Run("falls back to result when no concernedItem", func(t *testing.T) {
		d := newSnapRD(t)
		funcs := vmSnapshotCRUDFuncs{
			create: func(ctx context.Context, vmID, name string) (string, error) { return "act", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return snapActivity("", snapTestSnapID), nil
			},
			read: okRead,
		}
		if diags := createVMSnapshotWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Id() != snapTestSnapID {
			t.Fatalf("id = %q, want %q (from result)", d.Id(), snapTestSnapID)
		}
	})

	t.Run("no usable id fails closed", func(t *testing.T) {
		for _, act := range []*client.Activity{snapActivity("", ""), snapActivity("", snapTestVMID), snapActivity("", "not-a-uuid")} {
			d := newSnapRD(t)
			a := act
			funcs := vmSnapshotCRUDFuncs{
				create:       func(ctx context.Context, vmID, name string) (string, error) { return "act", nil },
				waitActivity: func(ctx context.Context, s string) (*client.Activity, error) { return a, nil },
			}
			if diags := createVMSnapshotWith(context.Background(), d, funcs); !diags.HasError() {
				t.Fatalf("activity %+v must fail closed", a)
			}
			if d.Id() != "" {
				t.Fatal("no id must be set on an ambiguous activity")
			}
		}
	})

	t.Run("wait failure fails without SetId", func(t *testing.T) {
		d := newSnapRD(t)
		funcs := vmSnapshotCRUDFuncs{
			create:       func(ctx context.Context, vmID, name string) (string, error) { return "act", nil },
			waitActivity: func(ctx context.Context, s string) (*client.Activity, error) { return nil, errors.New("failed") },
		}
		if diags := createVMSnapshotWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a wait failure must fail")
		}
		if d.Id() != "" {
			t.Fatal("a wait failure must not set an id")
		}
	})
}

func TestReadVMSnapshotInto(t *testing.T) {
	setup := func(t *testing.T) *schema.ResourceData {
		d := newSnapRD(t)
		d.SetId(snapTestSnapID)
		return d
	}
	nilSnap := func(ctx context.Context, vmID, snapID string) (*client.PublicCloudVMSnapshot, error) { return nil, nil }

	t.Run("found sets state", func(t *testing.T) {
		d := setup(t)
		funcs := vmSnapshotCRUDFuncs{read: func(ctx context.Context, vmID, snapID string) (*client.PublicCloudVMSnapshot, error) {
			return snapObj(snapID), nil
		}}
		if diags := readVMSnapshotInto(context.Background(), d, funcs, vmSnapshotReadForRefresh); diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if d.Get("status").(string) != "available" {
			t.Fatal("state not set")
		}
	})

	t.Run("read error keeps", func(t *testing.T) {
		d := setup(t)
		funcs := vmSnapshotCRUDFuncs{read: func(ctx context.Context, vmID, snapID string) (*client.PublicCloudVMSnapshot, error) {
			return nil, errors.New("403")
		}}
		if diags := readVMSnapshotInto(context.Background(), d, funcs, vmSnapshotReadForRefresh); !diags.HasError() {
			t.Fatal("read error must fail closed")
		}
		if d.Id() == "" {
			t.Fatal("must not drop")
		}
	})

	t.Run("afterWrite nil keeps id", func(t *testing.T) {
		d := setup(t)
		funcs := vmSnapshotCRUDFuncs{read: nilSnap}
		if diags := readVMSnapshotInto(context.Background(), d, funcs, vmSnapshotReadAfterWrite); !diags.HasError() {
			t.Fatal("afterWrite nil must fail closed")
		}
		if d.Id() == "" {
			t.Fatal("afterWrite must not drop")
		}
	})

	t.Run("refresh nil + VM gone drops", func(t *testing.T) {
		d := setup(t)
		funcs := vmSnapshotCRUDFuncs{
			read:   nilSnap,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
				return []*client.PublicCloudVMInstance{{ID: "other"}}, nil
			},
		}
		if diags := readVMSnapshotInto(context.Background(), d, funcs, vmSnapshotReadForRefresh); diags.HasError() {
			t.Fatalf("a gone VM must drop the snapshot, got %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("snapshot of a gone VM must be dropped")
		}
	})

	t.Run("refresh nil + VM still listed keeps", func(t *testing.T) {
		d := setup(t)
		funcs := vmSnapshotCRUDFuncs{
			read:   nilSnap,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) { return nil, nil },
			vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
				return []*client.PublicCloudVMInstance{{ID: snapTestVMID}}, nil
			},
		}
		if diags := readVMSnapshotInto(context.Background(), d, funcs, vmSnapshotReadForRefresh); !diags.HasError() {
			t.Fatal("a still-listed VM must keep the snapshot")
		}
	})

	t.Run("refresh nil + VM exists + snapshot absent drops", func(t *testing.T) {
		d := setup(t)
		funcs := vmSnapshotCRUDFuncs{
			read: nilSnap,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMSnapshot, error) {
				return []*client.PublicCloudVMSnapshot{{ID: "other"}}, nil
			},
		}
		if diags := readVMSnapshotInto(context.Background(), d, funcs, vmSnapshotReadForRefresh); diags.HasError() {
			t.Fatalf("an absent snapshot must drop, got %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("absent snapshot must be dropped")
		}
	})

	t.Run("refresh nil + snapshot still listed keeps", func(t *testing.T) {
		d := setup(t)
		funcs := vmSnapshotCRUDFuncs{
			read: nilSnap,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMSnapshot, error) {
				return []*client.PublicCloudVMSnapshot{{ID: snapTestSnapID}}, nil
			},
		}
		if diags := readVMSnapshotInto(context.Background(), d, funcs, vmSnapshotReadForRefresh); !diags.HasError() {
			t.Fatal("a still-listed snapshot must keep")
		}
	})

	t.Run("refresh nil + snapshot list fails keeps", func(t *testing.T) {
		d := setup(t)
		funcs := vmSnapshotCRUDFuncs{
			read: nilSnap,
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMSnapshot, error) {
				return nil, errors.New("total mismatch")
			},
		}
		if diags := readVMSnapshotInto(context.Background(), d, funcs, vmSnapshotReadForRefresh); !diags.HasError() {
			t.Fatal("a failed snapshot listing must keep")
		}
	})
}

func TestDeleteVMSnapshotWith(t *testing.T) {
	t.Run("snapshot deleted", func(t *testing.T) {
		d := newSnapRD(t)
		d.SetId(snapTestSnapID)
		deleted := false
		funcs := vmSnapshotCRUDFuncs{
			del: func(ctx context.Context, vmID, snapID string) (string, error) { deleted = true; return "act", nil },
			waitActivity: func(ctx context.Context, a string) (*client.Activity, error) {
				return snapActivity("", snapTestVMID), nil
			},
		}
		if diags := deleteVMSnapshotWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("delete: %v", diags)
		}
		if !deleted {
			t.Fatal("delete must call the API")
		}
	})

	t.Run("404 race but snapshot still listed keeps", func(t *testing.T) {
		d := newSnapRD(t)
		d.SetId(snapTestSnapID)
		funcs := vmSnapshotCRUDFuncs{
			del: func(ctx context.Context, vmID, snapID string) (string, error) {
				return "", client.StatusError{Code: 404}
			},
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMSnapshot, error) {
				return []*client.PublicCloudVMSnapshot{{ID: snapTestSnapID}}, nil
			},
		}
		if diags := deleteVMSnapshotWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a 404 while still listed must keep + error (never a bare-404 drop)")
		}
		if d.Id() == "" {
			t.Fatal("must not drop on an unconfirmed 404")
		}
	})

	t.Run("404 race confirmed absent drops", func(t *testing.T) {
		d := newSnapRD(t)
		d.SetId(snapTestSnapID)
		funcs := vmSnapshotCRUDFuncs{
			del: func(ctx context.Context, vmID, snapID string) (string, error) {
				return "", client.StatusError{Code: 404}
			},
			vmRead: func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error) {
				return &client.PublicCloudVMInstance{ID: vmID}, nil
			},
			listStrict: func(ctx context.Context, vmID string) ([]*client.PublicCloudVMSnapshot, error) {
				return []*client.PublicCloudVMSnapshot{}, nil
			},
		}
		if diags := deleteVMSnapshotWith(context.Background(), d, funcs); diags.HasError() {
			t.Fatalf("a 404 with confirmed absence must succeed, got %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("confirmed-absent snapshot must drop")
		}
	})

	t.Run("non-404 delete error keeps", func(t *testing.T) {
		d := newSnapRD(t)
		d.SetId(snapTestSnapID)
		funcs := vmSnapshotCRUDFuncs{
			del: func(ctx context.Context, vmID, snapID string) (string, error) {
				return "", client.StatusError{Code: 403}
			},
		}
		if diags := deleteVMSnapshotWith(context.Background(), d, funcs); !diags.HasError() {
			t.Fatal("a 403 delete error must fail closed")
		}
	})
}
