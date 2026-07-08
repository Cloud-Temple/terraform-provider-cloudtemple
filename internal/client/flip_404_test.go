package client

import (
	"context"
	"net/http"
	"testing"
)

// TestNotFound404Flip pins the #384 PR-A flip on the Backup and Marketplace reads:
// they now use requireNotFoundOrOK(resp, 404), so a 404 (absent) yields (nil, nil)
// while a 403 (forbidden) is surfaced as an "access denied" error instead of being
// masked as not-found. (VPC.Read has the same assertions in vpc_unit_test.go.)
//
// Mutation proof: reverting either site to notFoundCode=403 makes its 403 subtest
// RED (a 403 would again return (nil, nil) instead of an error).
func TestNotFound404Flip(t *testing.T) {
	notFound := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }
	forbidden := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }
	ctx := context.Background()

	t.Run("backup SLAPolicy.Read 404 -> (nil,nil)", func(t *testing.T) {
		c := newVPCTestClient(t, notFound)
		p, err := c.Backup().SLAPolicy().Read(ctx, "absent")
		if err != nil || p != nil {
			t.Fatalf("404 must yield (nil,nil); got policy=%+v err=%v", p, err)
		}
	})
	t.Run("backup SLAPolicy.Read 403 -> access-denied error", func(t *testing.T) {
		c := newVPCTestClient(t, forbidden)
		p, err := c.Backup().SLAPolicy().Read(ctx, "forbidden")
		if err == nil || p != nil {
			t.Fatalf("403 must yield an access-denied error and nil; got policy=%+v err=%v", p, err)
		}
	})
	t.Run("marketplace Item.Read 404 -> (nil,nil)", func(t *testing.T) {
		c := newVPCTestClient(t, notFound)
		it, err := c.Marketplace().Item().Read(ctx, "absent")
		if err != nil || it != nil {
			t.Fatalf("404 must yield (nil,nil); got item=%+v err=%v", it, err)
		}
	})
	t.Run("marketplace Item.Read 403 -> access-denied error", func(t *testing.T) {
		c := newVPCTestClient(t, forbidden)
		it, err := c.Marketplace().Item().Read(ctx, "forbidden")
		if err == nil || it != nil {
			t.Fatalf("403 must yield an access-denied error and nil; got item=%+v err=%v", it, err)
		}
	})
}
