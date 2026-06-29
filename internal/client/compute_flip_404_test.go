package client

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// TestComputeNotFound404Flip pins the #384 PR-B flip across the Compute reads
// (VMware + OpenIaaS): each now uses requireNotFoundOrOK(resp, 404), so a 404
// (absent) yields (nil,nil)/empty while a genuine 403 (forbidden) surfaces as an
// "access denied" error instead of being masked as not-found/empty. It covers
// BOTH categories the flip touches: by-id single-object Reads AND List/special
// endpoints.
//
// Mutation proof: reverting any covered site to notFoundCode=403 makes its 403
// subtest RED (a 403 would again return (nil,nil)/empty instead of erroring). The
// exhaustive guard in TestComputeReadsUseNotFound404 additionally fails if ANY of
// the 39 compute sites is reverted, whether exercised here or not.
func TestComputeNotFound404Flip(t *testing.T) {
	notFound := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }
	forbidden := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }
	ctx := context.Background()

	// assertAccessDenied checks a 403 now surfaces the specific access-denied
	// error (not merely "some error"): the flip's whole point is that a genuine
	// 403 is no longer masked.
	assertAccessDenied := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("a 403 must yield an access-denied error, got nil")
		}
		if !strings.Contains(err.Error(), "access denied") {
			t.Fatalf("a 403 must surface as an access-denied error, got: %v", err)
		}
	}

	// --- by-id single-object reads: 404 -> (nil,nil); 403 -> access-denied error ---

	t.Run("VMware VirtualMachine.Read 404 -> (nil,nil)", func(t *testing.T) {
		c := newPATTestClient(t, notFound)
		vm, err := c.Compute().VirtualMachine().Read(ctx, "absent")
		if err != nil || vm != nil {
			t.Fatalf("404 must yield (nil,nil); got vm=%+v err=%v", vm, err)
		}
	})
	t.Run("VMware VirtualMachine.Read 403 -> access-denied error", func(t *testing.T) {
		c := newPATTestClient(t, forbidden)
		vm, err := c.Compute().VirtualMachine().Read(ctx, "forbidden")
		if vm != nil {
			t.Fatalf("403 must yield a nil vm, got %+v", vm)
		}
		assertAccessDenied(t, err)
	})

	t.Run("OpenIaaS VirtualMachine.Read 404 -> (nil,nil)", func(t *testing.T) {
		c := newPATTestClient(t, notFound)
		vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, "absent")
		if err != nil || vm != nil {
			t.Fatalf("404 must yield (nil,nil); got vm=%+v err=%v", vm, err)
		}
	})
	t.Run("OpenIaaS VirtualMachine.Read 403 -> access-denied error", func(t *testing.T) {
		c := newPATTestClient(t, forbidden)
		vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, "forbidden")
		if vm != nil {
			t.Fatalf("403 must yield a nil vm, got %+v", vm)
		}
		assertAccessDenied(t, err)
	})

	t.Run("VMware VirtualDisk.Read 404 -> (nil,nil)", func(t *testing.T) {
		c := newPATTestClient(t, notFound)
		d, err := c.Compute().VirtualDisk().Read(ctx, "absent")
		if err != nil || d != nil {
			t.Fatalf("404 must yield (nil,nil); got disk=%+v err=%v", d, err)
		}
	})
	t.Run("VMware VirtualDisk.Read 403 -> access-denied error", func(t *testing.T) {
		c := newPATTestClient(t, forbidden)
		d, err := c.Compute().VirtualDisk().Read(ctx, "forbidden")
		if d != nil {
			t.Fatalf("403 must yield a nil disk, got %+v", d)
		}
		assertAccessDenied(t, err)
	})

	// --- list / special endpoints: 404 -> empty (no error); 403 -> access-denied error ---

	t.Run("VMware VirtualDisk.List 404 -> empty, no error", func(t *testing.T) {
		c := newPATTestClient(t, notFound)
		disks, err := c.Compute().VirtualDisk().List(ctx, &VirtualDiskFilter{VirtualMachineID: "vm-1"})
		if err != nil || len(disks) != 0 {
			t.Fatalf("404 must yield an empty listing and no error; got disks=%v err=%v", disks, err)
		}
	})
	t.Run("VMware VirtualDisk.List 403 -> access-denied error", func(t *testing.T) {
		c := newPATTestClient(t, forbidden)
		_, err := c.Compute().VirtualDisk().List(ctx, &VirtualDiskFilter{VirtualMachineID: "vm-1"})
		assertAccessDenied(t, err)
	})

	t.Run("OpenIaaS NetworkAdapter.List 404 -> empty, no error", func(t *testing.T) {
		c := newPATTestClient(t, notFound)
		ads, err := c.Compute().OpenIaaS().NetworkAdapter().List(ctx, &OpenIaaSNetworkAdapterFilter{VirtualMachineID: "vm-1"})
		if err != nil || len(ads) != 0 {
			t.Fatalf("404 must yield an empty listing and no error; got adapters=%v err=%v", ads, err)
		}
	})
	t.Run("OpenIaaS NetworkAdapter.List 403 -> access-denied error", func(t *testing.T) {
		c := newPATTestClient(t, forbidden)
		_, err := c.Compute().OpenIaaS().NetworkAdapter().List(ctx, &OpenIaaSNetworkAdapterFilter{VirtualMachineID: "vm-1"})
		assertAccessDenied(t, err)
	})

	t.Run("VMware VirtualMachine.Recommendation 404 -> empty, no error", func(t *testing.T) {
		c := newPATTestClient(t, notFound)
		recs, err := c.Compute().VirtualMachine().Recommendation(ctx, &VirtualMachineRecommendationFilter{})
		if err != nil || len(recs) != 0 {
			t.Fatalf("404 must yield an empty listing and no error; got recs=%v err=%v", recs, err)
		}
	})
	t.Run("VMware VirtualMachine.Recommendation 403 -> access-denied error", func(t *testing.T) {
		c := newPATTestClient(t, forbidden)
		_, err := c.Compute().VirtualMachine().Recommendation(ctx, &VirtualMachineRecommendationFilter{})
		assertAccessDenied(t, err)
	})
}
