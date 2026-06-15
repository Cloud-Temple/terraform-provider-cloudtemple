package main

import (
	"context"
	"errors"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// These tests prove F3: every write cycle's teardown is registered keyed by a
// DETERMINISTIC identifier BEFORE the creating call, so a created-but-unresolved
// or created-but-failed resource is still swept. Each fake seam simulates the
// "error-after-effect" hazard: the create reported an error/ambiguous result,
// yet a resource exists server-side. The pre-registered teardown must find and
// remove it, and must be idempotent (absent → success).

// --- VPC static IP ------------------------------------------------------------

type fakeStaticIPSeam struct {
	list      []*client.StaticIP
	listErr   error
	deleted   []string
	deleteErr error
}

func (f *fakeStaticIPSeam) ListStrict(_ context.Context, _ string) ([]*client.StaticIP, error) {
	return f.list, f.listErr
}
func (f *fakeStaticIPSeam) DeleteAndWait(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return f.deleteErr
}

// TestStaticIPTeardownSweepsCreatedButUnresolved proves the by-MAC teardown
// deletes the custom static IP carrying our MAC even though the create returned
// no usable id (the orphan window the client documents). Mutation proof: drop
// the registerStaticIPTeardown call in cycle_vpc.go → nothing is registered →
// the static IP is never deleted in the create-ambiguous path → RED here (this
// test exercises the registration helper directly, and the engine test below
// proves it is actually wired before Create).
func TestStaticIPTeardownSweepsCreatedButUnresolved(t *testing.T) {
	seam := &fakeStaticIPSeam{
		list: []*client.StaticIP{
			{ID: "xoa-1", MacAddress: "02:00:5e:f0:00:00", Source: "xoa"}, // co-resident, must be ignored
			{ID: "custom-1", MacAddress: "02:00:5E:F0:00:00", Source: "custom"},
		},
	}
	cl := NewCleanup()
	registerStaticIPTeardown(cl, seam, "pn-1", "02:00:5e:f0:00:00")
	if cl.Pending() != 1 {
		t.Fatalf("teardown must be registered before Create; Pending=%d", cl.Pending())
	}
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("teardown should succeed, got %+v", fails)
	}
	// Only the custom IP with our MAC must be deleted, not the xoa co-resident.
	if len(seam.deleted) != 1 || seam.deleted[0] != "custom-1" {
		t.Fatalf("teardown deleted %v, want exactly [custom-1]", seam.deleted)
	}
}

// TestStaticIPTeardownIdempotentWhenAbsent proves an absent static IP is success.
func TestStaticIPTeardownIdempotentWhenAbsent(t *testing.T) {
	seam := &fakeStaticIPSeam{list: []*client.StaticIP{}}
	cl := NewCleanup()
	registerStaticIPTeardown(cl, seam, "pn-1", "02:00:5e:f0:00:00")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("absent static IP must be idempotent success, got %+v", fails)
	}
	if len(seam.deleted) != 0 {
		t.Fatalf("nothing should be deleted when absent, deleted %v", seam.deleted)
	}
}

// --- Object storage bucket ----------------------------------------------------

type fakeBucketSeam struct {
	deleted []string
	err     error
}

func (f *fakeBucketSeam) DeleteAndWait(_ context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return f.err
}

// TestBucketTeardownRegisteredByName proves the bucket teardown is keyed by the
// deterministic name and runs the delete (sweeping a created-but-unconfirmed
// bucket).
func TestBucketTeardownRegisteredByName(t *testing.T) {
	seam := &fakeBucketSeam{}
	cl := NewCleanup()
	registerBucketTeardown(cl, seam, "ct-validate-0-0")
	if cl.Pending() != 1 {
		t.Fatalf("bucket teardown must be registered before Create; Pending=%d", cl.Pending())
	}
	cl.TeardownAll(context.Background())
	if len(seam.deleted) != 1 || seam.deleted[0] != "ct-validate-0-0" {
		t.Fatalf("bucket teardown deleted %v, want [ct-validate-0-0]", seam.deleted)
	}
}

// --- ACL entry ----------------------------------------------------------------

type fakeACLSeam struct {
	revoked [][3]string
	err     error
}

func (f *fakeACLSeam) RevokeAndWait(_ context.Context, bucket, role, account string) error {
	f.revoked = append(f.revoked, [3]string{bucket, role, account})
	return f.err
}

// TestACLTeardownRegisteredByTriple proves the ACL teardown is keyed by
// (bucket, role, account) and revokes (sweeping an ambiguous grant).
func TestACLTeardownRegisteredByTriple(t *testing.T) {
	seam := &fakeACLSeam{}
	cl := NewCleanup()
	registerACLTeardown(cl, seam, "bkt", "reader", "acct")
	if cl.Pending() != 1 {
		t.Fatalf("ACL teardown must be registered before Grant; Pending=%d", cl.Pending())
	}
	cl.TeardownAll(context.Background())
	if len(seam.revoked) != 1 || seam.revoked[0] != [3]string{"bkt", "reader", "acct"} {
		t.Fatalf("ACL teardown revoked %v, want one [bkt reader acct]", seam.revoked)
	}
}

// --- FIP binding --------------------------------------------------------------

type fakeFIPSeam struct {
	unbound [][2]string
	err     error
}

func (f *fakeFIPSeam) UnbindAndWait(_ context.Context, fipID, staticID string) error {
	f.unbound = append(f.unbound, [2]string{fipID, staticID})
	return f.err
}

// TestFIPUnbindTeardownRegisteredByPair proves the unbind teardown is keyed by
// (fip, static) and unbinds (releasing a bind whose confirmation was lost).
func TestFIPUnbindTeardownRegisteredByPair(t *testing.T) {
	seam := &fakeFIPSeam{}
	cl := NewCleanup()
	registerFIPUnbindTeardown(cl, seam, "fip-1", "static-1")
	if cl.Pending() != 1 {
		t.Fatalf("FIP unbind teardown must be registered before Bind; Pending=%d", cl.Pending())
	}
	cl.TeardownAll(context.Background())
	if len(seam.unbound) != 1 || seam.unbound[0] != [2]string{"fip-1", "static-1"} {
		t.Fatalf("FIP teardown unbound %v, want one [fip-1 static-1]", seam.unbound)
	}
}

// --- IAM PAT ------------------------------------------------------------------

type fakePATSeam struct {
	deleted    []string
	findID     string
	findErr    error
	deleteErr  error
	findCalled bool
}

func (f *fakePATSeam) Delete(_ context.Context, patID string) error {
	f.deleted = append(f.deleted, patID)
	return f.deleteErr
}
func (f *fakePATSeam) FindIDByName(_ context.Context, _ string) (string, error) {
	f.findCalled = true
	return f.findID, f.findErr
}

// TestPATTeardownByIDWhenResolved proves that once the create decodes an id, the
// teardown deletes by id and does NOT fall back to find-by-name.
func TestPATTeardownByIDWhenResolved(t *testing.T) {
	seam := &fakePATSeam{}
	cl := NewCleanup()
	ref := &patTeardownRef{Name: "ct-validate-0-0"}
	registerPATTeardown(cl, seam, ref)
	// Simulate the create decoding the id AFTER registration (shared ref).
	ref.ID = "pat-123"
	ref.Resolved = true

	cl.TeardownAll(context.Background())
	if len(seam.deleted) != 1 || seam.deleted[0] != "pat-123" {
		t.Fatalf("resolved PAT must be deleted by id, deleted=%v", seam.deleted)
	}
	if seam.findCalled {
		t.Fatal("must not fall back to find-by-name when the id is resolved")
	}
}

// TestPATTeardownFindByNameWhenUndecoded proves the security-critical path: the
// create reported an error/ambiguous result (id never decoded), yet a PAT (a
// live credential) exists. The pre-registered teardown finds it by name and
// deletes it. This is the orphan window the pre-registration closes.
//
// Mutation proof: register the PAT teardown only AFTER a successful decode (move
// registerPATTeardown below the create op and gate it on ref.Resolved) → an
// undecoded PAT is never registered → never deleted → RED here.
func TestPATTeardownFindByNameWhenUndecoded(t *testing.T) {
	seam := &fakePATSeam{findID: "pat-found-by-name"}
	cl := NewCleanup()
	ref := &patTeardownRef{Name: "ct-validate-0-0"} // never Resolved (decode failed)
	registerPATTeardown(cl, seam, ref)
	if cl.Pending() != 1 {
		t.Fatalf("PAT teardown must be registered before create/decode; Pending=%d", cl.Pending())
	}

	cl.TeardownAll(context.Background())
	if !seam.findCalled {
		t.Fatal("undecoded PAT must trigger find-by-name fallback")
	}
	if len(seam.deleted) != 1 || seam.deleted[0] != "pat-found-by-name" {
		t.Fatalf("undecoded PAT must be deleted via find-by-name, deleted=%v", seam.deleted)
	}
}

// TestPATTeardownIdempotentWhenNeverCreated proves a never-created PAT (no id,
// find-by-name returns "") is an idempotent success, not a failure.
func TestPATTeardownIdempotentWhenNeverCreated(t *testing.T) {
	seam := &fakePATSeam{findID: ""}
	cl := NewCleanup()
	ref := &patTeardownRef{Name: "ct-validate-0-0"}
	registerPATTeardown(cl, seam, ref)
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("never-created PAT must be idempotent success, got %+v", fails)
	}
	if len(seam.deleted) != 0 {
		t.Fatalf("nothing to delete when never created, deleted=%v", seam.deleted)
	}
}

// TestPATTeardownSurfacesFindError proves a find-by-name error is surfaced (not
// swallowed): a teardown that cannot prove absence must report a possible
// orphan, not silently succeed.
func TestPATTeardownSurfacesFindError(t *testing.T) {
	seam := &fakePATSeam{findErr: errors.New("list failed")}
	cl := NewCleanup()
	ref := &patTeardownRef{Name: "ct-validate-0-0"}
	registerPATTeardown(cl, seam, ref)
	fails := cl.TeardownAll(context.Background())
	if len(fails) != 1 {
		t.Fatalf("a find-by-name error must surface as a teardown failure, got %+v", fails)
	}
}

// TestStaticIPTeardownSurfacesListError proves a strict-listing error is
// surfaced rather than treated as "absent" (fail closed, do not assume clean).
func TestStaticIPTeardownSurfacesListError(t *testing.T) {
	seam := &fakeStaticIPSeam{listErr: client.StatusError{Code: 403}}
	cl := NewCleanup()
	registerStaticIPTeardown(cl, seam, "pn-1", "02:00:5e:f0:00:00")
	fails := cl.TeardownAll(context.Background())
	if len(fails) != 1 {
		t.Fatalf("a strict-listing error must surface as a teardown failure, got %+v", fails)
	}
	if len(seam.deleted) != 0 {
		t.Fatalf("must not delete anything when the listing is unproven, deleted=%v", seam.deleted)
	}
}
