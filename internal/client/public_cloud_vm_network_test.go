package client

import (
	"context"
	"net/http"
	"testing"
)

func TestPublicCloudVMNetworkList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/networks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		// Bare array (verified live) — no wrapper, no total, no vpc block.
		_, _ = w.Write([]byte(`[{"id":"net-1","name":"LAN01"},{"id":"net-2","name":"test222"}]`))
	})
	nets, err := c.PublicCloudVM().Network().List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(nets) != 2 {
		t.Fatalf("want 2 networks, got %d", len(nets))
	}
	if nets[0].ID != "net-1" || nets[0].Name != "LAN01" || nets[1].ID != "net-2" {
		t.Fatalf("networks not decoded: %+v", nets)
	}
}

func TestPublicCloudVMNetworkListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 returns the catalogue", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"net-1","name":"LAN01"}]`))
		})
		nets, err := c.PublicCloudVM().Network().ListStrict(ctx)
		if err != nil || len(nets) != 1 {
			t.Fatalf("complete listing: err=%v nets=%d", err, len(nets))
		}
	})

	t.Run("206 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte(`[]`))
		})
		if _, err := c.PublicCloudVM().Network().ListStrict(ctx); err == nil {
			t.Fatal("a 206 must fail closed for ListStrict")
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		if _, err := c.PublicCloudVM().Network().ListStrict(ctx); err == nil {
			t.Fatal("a 403 must fail closed")
		}
	})
}

// TestPublicCloudVMNetworkRead pins the read-only data source contract: absence
// on this endpoint is NOT a clean 404 (the live platform returns 400/500), so any
// non-200 fails closed with an error rather than a (nil,nil) "absent" signal.
func TestPublicCloudVMNetworkRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 decodes", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/networks/net-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"net-1","name":"LAN01"}`))
		})
		net, err := c.PublicCloudVM().Network().Read(ctx, "net-1")
		if err != nil || net == nil || net.Name != "LAN01" {
			t.Fatalf("bad network: %+v err=%v", net, err)
		}
	})

	// 400/500 = live absence codes; 404 = not a clean absence here either; 206/201
	// must NOT be decoded as success on a read-only single lookup.
	for _, code := range []int{http.StatusBadRequest, http.StatusInternalServerError, http.StatusNotFound, http.StatusPartialContent, http.StatusCreated} {
		code := code
		t.Run(http.StatusText(code)+" fails closed", func(t *testing.T) {
			c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) })
			if _, err := c.PublicCloudVM().Network().Read(ctx, "missing"); err == nil {
				t.Fatalf("HTTP %d must fail closed (never a silent absent) for a read-only network lookup", code)
			}
		})
	}
}
