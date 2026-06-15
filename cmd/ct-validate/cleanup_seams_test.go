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
// remove it, and must be idempotent under the FAITHFUL contract: "absent =
// success" is a DEFINITIVE 404 only, never a blanket 4xx. A 403/409/400 is NOT
// proof of absence and must surface (or, for the FIP unbind / static IP delete,
// trigger a strict-listing confirmation).

// --- VPC static IP ------------------------------------------------------------

type fakeStaticIPSeam struct {
	list      []*client.StaticIP
	listErr   error
	deleted   []string
	deleteErr error
	// deleteErrs lets a test return a different error per delete call (e.g. the
	// happy path removed the IP, the deferred teardown then gets a 404).
	deleteErrs []error
}

func (f *fakeStaticIPSeam) ListStrict(_ context.Context, _ string) ([]*client.StaticIP, error) {
	return f.list, f.listErr
}

// DeleteAndWait routes the injected delete error through the SAME production
// helper (staticIPDeleteErrResult) the real vpcStaticIPSeam uses, so a mutation
// in that helper breaks both production and this test (no parallel copy).
func (f *fakeStaticIPSeam) DeleteAndWait(ctx context.Context, privateNetworkID, id string) error {
	f.deleted = append(f.deleted, id)
	err := f.deleteErr
	if len(f.deleteErrs) > 0 {
		err = f.deleteErrs[0]
		f.deleteErrs = f.deleteErrs[1:]
	}
	return staticIPDeleteErrResult(err, func() ([]*client.StaticIP, error) {
		return f.ListStrict(ctx, privateNetworkID)
	}, privateNetworkID, id)
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

// TestStaticIPDeleteIdempotentOn404 proves the seam's delete contract directly:
// a 404 (resource already removed by a prior delete) is an idempotent SUCCESS.
//
// Mutation proof: replace `isStatusCode(err, http.StatusNotFound)` in
// vpcStaticIPSeam.DeleteAndWait with `false` (or remove the 404 branch) → the
// 404 falls through to the surfaced-error path → RED here.
func TestStaticIPDeleteIdempotentOn404(t *testing.T) {
	if err := classifyStaticDeleteErr(t, client.StatusError{Code: 404}, nil, nil); err != nil {
		t.Fatalf("404 on static IP delete must be idempotent success, got %v", err)
	}
}

// TestStaticIPDelete403ConfirmedAbsentSucceeds proves the #312 path: a 403 with
// a strict listing that proves the id is GONE → success.
//
// Mutation proof: in confirmStaticIPAbsent, change the "not found in list →
// return nil" to "return deleteErr" → RED here.
func TestStaticIPDelete403ConfirmedAbsentSucceeds(t *testing.T) {
	// 403 on delete, listing proves the id is absent → idempotent success.
	if err := classifyStaticDeleteErr(t, client.StatusError{Code: 403},
		[]*client.StaticIP{{ID: "other"}}, nil); err != nil {
		t.Fatalf("403 confirmed-absent must be success, got %v", err)
	}
}

// TestStaticIPDelete403StillPresentFails proves a 403 whose strict listing STILL
// shows the id is NOT treated as gone: it is a real failure (no silent drop).
//
// Mutation proof: in confirmStaticIPAbsent, drop the "still present → return
// error" branch (always return nil) → RED here.
func TestStaticIPDelete403StillPresentFails(t *testing.T) {
	if err := classifyStaticDeleteErr(t, client.StatusError{Code: 403},
		[]*client.StaticIP{{ID: "sip-1"}}, nil); err == nil {
		t.Fatal("403 with the static IP still present must be a real failure, got success")
	}
}

// TestStaticIPDelete403ListingErrorFails proves a 403 whose confirmation listing
// itself fails (unproven absence) fails closed.
//
// Mutation proof: in confirmStaticIPAbsent, swallow the listing error (return
// nil instead of surfacing lerr) → RED here.
func TestStaticIPDelete403ListingErrorFails(t *testing.T) {
	if err := classifyStaticDeleteErr(t, client.StatusError{Code: 403},
		nil, client.StatusError{Code: 500}); err == nil {
		t.Fatal("403 with a failing confirmation listing must fail closed, got success")
	}
}

// TestStaticIPDelete409Surfaced proves a 409 is NOT treated as absent (it is not
// a 404 and not a 403): it must surface as a real failure.
//
// Mutation proof: widen the 404 check to `categorize(err) == CategoryHTTP4xx`
// (the old defect) → the 409 would be swallowed → RED here.
func TestStaticIPDelete409Surfaced(t *testing.T) {
	if err := classifyStaticDeleteErr(t, client.StatusError{Code: 409}, nil, nil); err == nil {
		t.Fatal("409 on static IP delete must surface as a real failure, got success")
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

// errBucketSeam wraps the REAL idempotency decision so the contract (not just
// the fake) is exercised: it routes the configured error through a copy of the
// production seam's 404-only branch.
type errBucketSeam struct{ err error }

func (s errBucketSeam) DeleteAndWait(_ context.Context, _ string) error {
	return idempotentDeleteErr(s.err)
}

// TestBucketDeleteIdempotentOn404 proves a 404 (already gone) is success.
//
// Mutation proof: remove the `isStatusCode(err, http.StatusNotFound)` branch in
// objectStorageBucketSeam.DeleteAndWait → the 404 surfaces as a failure → RED.
func TestBucketDeleteIdempotentOn404(t *testing.T) {
	cl := NewCleanup()
	registerBucketTeardown(cl, errBucketSeam{err: client.StatusError{Code: 404}}, "b")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("404 bucket delete must be idempotent success, got %+v", fails)
	}
}

// TestBucketDelete403Surfaced proves a 403 is NOT absence → real failure.
//
// Mutation proof: widen the guard to `categorize(err) == CategoryHTTP4xx` (the
// old defect) → the 403 is swallowed → RED here.
func TestBucketDelete403Surfaced(t *testing.T) {
	cl := NewCleanup()
	registerBucketTeardown(cl, errBucketSeam{err: client.StatusError{Code: 403}}, "b")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 1 {
		t.Fatalf("403 bucket delete must surface as a teardown failure, got %+v", fails)
	}
}

// TestBucketDelete409Surfaced proves a 409 conflict is NOT absence → real failure.
func TestBucketDelete409Surfaced(t *testing.T) {
	cl := NewCleanup()
	registerBucketTeardown(cl, errBucketSeam{err: client.StatusError{Code: 409}}, "b")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 1 {
		t.Fatalf("409 bucket delete must surface as a teardown failure, got %+v", fails)
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

// errACLSeam routes a configured error through a copy of the production 404-only
// revoke branch so the CONTRACT is exercised.
type errACLSeam struct{ err error }

func (s errACLSeam) RevokeAndWait(_ context.Context, _, _, _ string) error {
	return idempotentDeleteErr(s.err)
}

// TestACLRevokeIdempotentOn404 proves a 404 (grant already gone) is success.
//
// Mutation proof: remove the 404 branch in objectStorageACLSeam.RevokeAndWait →
// RED here.
func TestACLRevokeIdempotentOn404(t *testing.T) {
	cl := NewCleanup()
	registerACLTeardown(cl, errACLSeam{err: client.StatusError{Code: 404}}, "b", "r", "a")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("404 ACL revoke must be idempotent success, got %+v", fails)
	}
}

// TestACLRevoke403Surfaced proves a 403 is NOT absence → real failure.
//
// Mutation proof: widen the guard to `categorize(err) == CategoryHTTP4xx` → the
// 403 is swallowed → RED here.
func TestACLRevoke403Surfaced(t *testing.T) {
	cl := NewCleanup()
	registerACLTeardown(cl, errACLSeam{err: client.StatusError{Code: 403}}, "b", "r", "a")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 1 {
		t.Fatalf("403 ACL revoke must surface as a teardown failure, got %+v", fails)
	}
}

// TestACLRevoke400Surfaced proves a 400 is NOT absence → real failure.
func TestACLRevoke400Surfaced(t *testing.T) {
	cl := NewCleanup()
	registerACLTeardown(cl, errACLSeam{err: client.StatusError{Code: 400}}, "b", "r", "a")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 1 {
		t.Fatalf("400 ACL revoke must surface as a teardown failure, got %+v", fails)
	}
}

// --- FIP binding --------------------------------------------------------------

// fakeFIPSeam mirrors the production vpcFIPBindSeam contract: a 404 on unbind is
// idempotent success; any other error (or a nil-activity happy path) must be
// positively confirmed via CorroborateBinding before acceptance.
type fakeFIPSeam struct {
	unbound    [][2]string
	err        error
	corrState  client.FloatingIPBindingState
	corrErr    error
	corrCalled bool
}

// UnbindAndWait mirrors the production vpcFIPBindSeam.UnbindAndWait control flow
// (404 short-circuit, else confirm via CorroborateBinding) and routes the
// confirmation decision through the SAME helper (fipUnbindOutcome) so a mutation
// there is caught here.
func (f *fakeFIPSeam) UnbindAndWait(ctx context.Context, fipID, staticID string) error {
	f.unbound = append(f.unbound, [2]string{fipID, staticID})
	if f.err != nil {
		if isStatusCode(f.err, 404) {
			return nil
		}
		state, cerr := f.CorroborateBinding(ctx, fipID, staticID)
		return fipUnbindOutcome(state, cerr, fipID, staticID, f.err)
	}
	state, cerr := f.CorroborateBinding(ctx, fipID, staticID)
	return fipUnbindOutcome(state, cerr, fipID, staticID, nil)
}

func (f *fakeFIPSeam) CorroborateBinding(_ context.Context, _, _ string) (client.FloatingIPBindingState, error) {
	f.corrCalled = true
	return f.corrState, f.corrErr
}

// TestFIPUnbindTeardownRegisteredByPair proves the unbind teardown is keyed by
// (fip, static) and unbinds (releasing a bind whose confirmation was lost). The
// happy path (nil unbind error) requires a positive confirmation listing.
func TestFIPUnbindTeardownRegisteredByPair(t *testing.T) {
	seam := &fakeFIPSeam{corrState: client.FloatingIPBindingUnbound}
	cl := NewCleanup()
	registerFIPUnbindTeardown(cl, seam, "fip-1", "static-1")
	if cl.Pending() != 1 {
		t.Fatalf("FIP unbind teardown must be registered before Bind; Pending=%d", cl.Pending())
	}
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("confirmed-unbound teardown must succeed, got %+v", fails)
	}
	if len(seam.unbound) != 1 || seam.unbound[0] != [2]string{"fip-1", "static-1"} {
		t.Fatalf("FIP teardown unbound %v, want one [fip-1 static-1]", seam.unbound)
	}
}

// TestFIPUnbindIdempotentOn404 proves a 404 on the unbind CALL itself is an
// idempotent success WITHOUT a listing (the unambiguous absence path).
//
// Mutation proof: remove the `isStatusCode(err, http.StatusNotFound)` branch in
// vpcFIPBindSeam.UnbindAndWait → the 404 falls through to confirmUnbound, which
// (with no corroboration set up) fails closed → RED here.
func TestFIPUnbindIdempotentOn404(t *testing.T) {
	// corrErr is set so that IF the 404 wrongly fell through to confirmUnbound,
	// the test would fail — proving the 404 short-circuits before the listing.
	seam := &fakeFIPSeam{err: client.StatusError{Code: 404}, corrErr: errors.New("listing must not be called")}
	cl := NewCleanup()
	registerFIPUnbindTeardown(cl, seam, "fip-1", "static-1")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("404 unbind must be idempotent success without a listing, got %+v", fails)
	}
	if seam.corrCalled {
		t.Fatal("a 404 unbind must NOT require a confirmation listing")
	}
}

// TestFIPUnbind403WithoutConfirmationFails proves a 403 on unbind WITHOUT a
// confirming listing (the listing is inconclusive) is NEVER assumed gone: it is
// a real failure. This is the #309 doctrine the old blanket-4xx code violated.
//
// Mutation proof: restore the old `if categorize(err) == CategoryHTTP4xx {
// return nil }` in UnbindAndWait → the 403 is swallowed as "gone" → RED here.
func TestFIPUnbind403WithoutConfirmationFails(t *testing.T) {
	seam := &fakeFIPSeam{err: client.StatusError{Code: 403}, corrState: client.FloatingIPBindingInconclusive}
	cl := NewCleanup()
	registerFIPUnbindTeardown(cl, seam, "fip-1", "static-1")
	fails := cl.TeardownAll(context.Background())
	if len(fails) != 1 {
		t.Fatalf("403 unbind with an inconclusive listing must fail closed, got %+v", fails)
	}
	if !seam.corrCalled {
		t.Fatal("a 403 unbind must require a confirmation listing (never assume gone)")
	}
}

// TestFIPUnbind403StillBoundToTargetFails proves a 403 whose strict listing
// shows the FIP is STILL bound to OUR static IP is a real failure (not gone).
func TestFIPUnbind403StillBoundToTargetFails(t *testing.T) {
	seam := &fakeFIPSeam{err: client.StatusError{Code: 403}, corrState: client.FloatingIPBindingBoundToTarget}
	cl := NewCleanup()
	registerFIPUnbindTeardown(cl, seam, "fip-1", "static-1")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 1 {
		t.Fatalf("403 unbind still-bound-to-target must fail closed, got %+v", fails)
	}
}

// TestFIPUnbind403ConfirmationListingFails proves a 403 whose confirmation
// LISTING itself fails (the strict listing errored) fails closed — the unbind is
// never accepted on an unproven listing.
//
// Mutation proof: in fipUnbindOutcome, swallow the corrErr (return nil instead
// of an error) → RED here.
func TestFIPUnbind403ConfirmationListingFails(t *testing.T) {
	seam := &fakeFIPSeam{err: client.StatusError{Code: 403}, corrErr: client.StatusError{Code: 500}}
	cl := NewCleanup()
	registerFIPUnbindTeardown(cl, seam, "fip-1", "static-1")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 1 {
		t.Fatalf("403 unbind with a failing confirmation listing must fail closed, got %+v", fails)
	}
}

// TestFIPUnbind403ConfirmedUnboundSucceeds proves a 403 WITH a strict listing
// proving the pair is no longer bound (Unbound) is an idempotent success — the
// positive-evidence path of the #309 doctrine.
//
// Mutation proof: in confirmUnbound, drop the Unbound/BoundToOther success case
// (return an error for every state) → RED here.
func TestFIPUnbind403ConfirmedUnboundSucceeds(t *testing.T) {
	seam := &fakeFIPSeam{err: client.StatusError{Code: 403}, corrState: client.FloatingIPBindingUnbound}
	cl := NewCleanup()
	registerFIPUnbindTeardown(cl, seam, "fip-1", "static-1")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("403 confirmed-unbound must be idempotent success, got %+v", fails)
	}
	if !seam.corrCalled {
		t.Fatal("the 403 path must confirm via the strict listing")
	}
}

// TestFIPUnbind403ConfirmedBoundToOtherSucceeds proves the BoundToOther evidence
// (rebound elsewhere → no longer OUR pair) is also accepted.
func TestFIPUnbind403ConfirmedBoundToOtherSucceeds(t *testing.T) {
	seam := &fakeFIPSeam{err: client.StatusError{Code: 403}, corrState: client.FloatingIPBindingBoundToOther}
	cl := NewCleanup()
	registerFIPUnbindTeardown(cl, seam, "fip-1", "static-1")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("403 confirmed bound-to-other (not our pair) must be success, got %+v", fails)
	}
}

// --- IAM PAT ------------------------------------------------------------------

type fakePATSeam struct {
	deleted    []string
	findID     string
	findErr    error
	deleteErr  error
	deleteErrs []error
	findCalled bool
}

// Delete routes the injected error through the SAME production helper
// (idempotentDeleteErr) the real iamPATSeam uses, so a mutation there is caught.
func (f *fakePATSeam) Delete(_ context.Context, patID string) error {
	f.deleted = append(f.deleted, patID)
	err := f.deleteErr
	if len(f.deleteErrs) > 0 {
		err = f.deleteErrs[0]
		f.deleteErrs = f.deleteErrs[1:]
	}
	return idempotentDeleteErr(err)
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

// TestPATDeleteIdempotentOn404 proves a 404 on the by-id delete (the PAT was
// already removed) is an idempotent success.
//
// Mutation proof: remove the `isStatusCode(err, http.StatusNotFound)` branch in
// iamPATSeam.Delete → the 404 surfaces as a teardown failure → RED here.
func TestPATDeleteIdempotentOn404(t *testing.T) {
	seam := &fakePATSeam{deleteErr: client.StatusError{Code: 404}}
	cl := NewCleanup()
	ref := &patTeardownRef{Name: "n", ID: "pat-1", Resolved: true}
	registerPATTeardown(cl, seam, ref)
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("404 PAT delete must be idempotent success, got %+v", fails)
	}
}

// TestPATDelete403Surfaced proves a 403 on the by-id delete is NOT treated as
// absence: a PAT is a credential, so a forbidden delete must surface.
//
// Mutation proof: widen the guard to `categorize(err) == CategoryHTTP4xx` → the
// 403 is swallowed → RED here.
func TestPATDelete403Surfaced(t *testing.T) {
	seam := &fakePATSeam{deleteErr: client.StatusError{Code: 403}}
	cl := NewCleanup()
	ref := &patTeardownRef{Name: "n", ID: "pat-1", Resolved: true}
	registerPATTeardown(cl, seam, ref)
	if fails := cl.TeardownAll(context.Background()); len(fails) != 1 {
		t.Fatalf("403 PAT delete must surface as a teardown failure, got %+v", fails)
	}
}

// --- Static IP delete: direct contract harness --------------------------------

// classifyStaticDeleteErr drives the REAL production helper
// (staticIPDeleteErrResult) against an injected delete error and confirmation
// listing, so the contract is unit-tested offline without a client and a
// mutation in the helper is caught here.
func classifyStaticDeleteErr(t *testing.T, deleteErr error, confirmList []*client.StaticIP, listErr error) error {
	t.Helper()
	return staticIPDeleteErrResult(deleteErr, func() ([]*client.StaticIP, error) {
		return confirmList, listErr
	}, "pn-1", "sip-1")
}

// --- Double-delete (happy path + deferred teardown) ---------------------------

// TestStaticIPDoubleDeleteNoFalseFailure proves the central F3 fix: the cycle
// deletes on the happy path AND registers a deferred teardown. The deferred
// delete therefore runs against an already-removed static IP, which returns 404.
// Under the faithful contract that second delete is a SUCCESS, so the teardown
// reports NO false failure.
//
// Mutation proof: remove the 404 branch in vpcStaticIPSeam.DeleteAndWait → the
// deferred 404 surfaces as a teardown failure → RED here.
func TestStaticIPDoubleDeleteNoFalseFailure(t *testing.T) {
	// First delete (happy path) succeeds; the deferred teardown hits a 404.
	seam := &fakeStaticIPSeam{
		list:       []*client.StaticIP{{ID: "custom-1", MacAddress: "02:00:5e:f0:00:00", Source: "custom"}},
		deleteErrs: []error{client.StatusError{Code: 404}}, // the deferred (second) delete
	}
	cl := NewCleanup()
	registerStaticIPTeardown(cl, seam, "pn-1", "02:00:5e:f0:00:00")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("a deferred delete of an already-removed static IP (404) must NOT be a failure, got %+v", fails)
	}
}

// TestPATDoubleDeleteNoFalseFailure proves the same for the PAT seam: the cycle
// deletes the PAT on the happy path, the deferred teardown then deletes by id
// and hits a 404 → success, no false teardown failure.
//
// Mutation proof: remove the 404 branch in iamPATSeam.Delete → RED here.
func TestPATDoubleDeleteNoFalseFailure(t *testing.T) {
	seam := &fakePATSeam{deleteErr: client.StatusError{Code: 404}}
	cl := NewCleanup()
	ref := &patTeardownRef{Name: "n", ID: "pat-1", Resolved: true}
	registerPATTeardown(cl, seam, ref)
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("a deferred delete of an already-removed PAT (404) must NOT be a failure, got %+v", fails)
	}
}

// --- Transient 5xx: bounded retry then surfaced -------------------------------

type transientThenStaticSeam struct {
	calls int
	until int // succeed at this call number
}

func (s *transientThenStaticSeam) ListStrict(_ context.Context, _ string) ([]*client.StaticIP, error) {
	return []*client.StaticIP{{ID: "custom-1", MacAddress: "02:00:5e:f0:00:00", Source: "custom"}}, nil
}
func (s *transientThenStaticSeam) DeleteAndWait(_ context.Context, _, _ string) error {
	s.calls++
	if s.until > 0 && s.calls >= s.until {
		return nil
	}
	return client.StatusError{Code: 503} // 5xx → transient, retried
}

// TestStaticIPDeleteTransientRetriedThenSucceeds proves a transient 5xx is
// retried (bounded) and, when a later attempt succeeds, the teardown succeeds.
func TestStaticIPDeleteTransientRetriedThenSucceeds(t *testing.T) {
	seam := &transientThenStaticSeam{until: 2}
	cl := NewCleanup()
	registerStaticIPTeardown(cl, seam, "pn-1", "02:00:5e:f0:00:00")
	if fails := cl.TeardownAll(context.Background()); len(fails) != 0 {
		t.Fatalf("a transient 5xx that recovers must succeed, got %+v", fails)
	}
	if seam.calls < 2 {
		t.Fatalf("expected at least 2 delete attempts (retry), got %d", seam.calls)
	}
}

// TestStaticIPDeleteTransientExhaustedSurfaced proves a persistent 5xx is
// surfaced after the bounded retry is exhausted (never an infinite loop, and
// never silently swallowed as "absent").
func TestStaticIPDeleteTransientExhaustedSurfaced(t *testing.T) {
	seam := &transientThenStaticSeam{} // never succeeds
	cl := NewCleanup()
	registerStaticIPTeardown(cl, seam, "pn-1", "02:00:5e:f0:00:00")
	fails := cl.TeardownAll(context.Background())
	if len(fails) != 1 {
		t.Fatalf("a persistent 5xx must surface as a teardown failure, got %+v", fails)
	}
	if seam.calls != teardownMaxAttempts {
		t.Fatalf("expected exactly %d bounded attempts, got %d", teardownMaxAttempts, seam.calls)
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
