package client

import (
	"context"
	"math"
	"net/http"
	"strings"
	"testing"
)

// TestPublicCloudVMStorageTypeList pins the WRAPPED-response decode
// ({ "storageTypes": [...] }) including the camelCase int/bool fields and the
// nested `sku` billing object (issue #507).
func TestPublicCloudVMStorageTypeList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/storage_types" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		// A nil filter must put NOTHING on the wire (backward-compatible list).
		if r.URL.RawQuery != "" {
			t.Errorf("nil filter must send no query params, got %q", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storageTypes":[{"id":"st-1","name":"Standard","description":"Standard block storage","iopsHint":"~1500 IOPS/TB","minSizeGb":1,"maxSizeGb":2048,"isAvailable":true,"sku":{"name":"csp:fr1:iaas:storage:bloc:medium:v1","price":0.063,"unit":"1 Gio","description":"Stockage bloc medium","descriptionEn":"Medium block storage"}}]}`))
	})
	sts, err := c.PublicCloudVM().StorageType().List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sts) != 1 {
		t.Fatalf("want 1 storage type, got %d", len(sts))
	}
	st := sts[0]
	if st.ID != "st-1" || st.Name != "Standard" || st.IopsHint != "~1500 IOPS/TB" {
		t.Fatalf("strings not decoded: %+v", st)
	}
	if st.MinSizeGb != 1 || st.MaxSizeGb != 2048 || !st.IsAvailable {
		t.Fatalf("numeric/bool fields not decoded: %+v", st)
	}
	if st.Sku == nil {
		t.Fatalf("sku not decoded: %+v", st)
	}
	if st.Sku.Name != "csp:fr1:iaas:storage:bloc:medium:v1" || st.Sku.Unit != "1 Gio" {
		t.Fatalf("sku string fields not decoded: %+v", st.Sku)
	}
	if st.Sku.Description != "Stockage bloc medium" || st.Sku.DescriptionEn != "Medium block storage" {
		t.Fatalf("sku description pair not decoded (fr/en spelling lock): %+v", st.Sku)
	}
	// Compare the parsed float with a tolerance rather than exact equality.
	if math.Abs(st.Sku.Price-0.063) > 1e-9 {
		t.Fatalf("sku price not decoded: got %v, want ~0.063", st.Sku.Price)
	}
}

// TestPublicCloudVMStorageTypeListSkuAbsent pins that a storage type whose `sku`
// is OMITTED or explicitly null decodes to a nil Sku pointer (never a phantom
// zero-priced object). This is the decode half of the pointer design: the
// flatten half turns a nil Sku into an empty list.
func TestPublicCloudVMStorageTypeListSkuAbsent(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storageTypes":[` +
			`{"id":"st-omit","name":"NoSku","minSizeGb":1,"maxSizeGb":10,"isAvailable":true},` +
			`{"id":"st-null","name":"NullSku","minSizeGb":1,"maxSizeGb":10,"isAvailable":true,"sku":null}` +
			`]}`))
	})
	sts, err := c.PublicCloudVM().StorageType().List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sts) != 2 {
		t.Fatalf("want 2 storage types, got %d", len(sts))
	}
	for _, st := range sts {
		if st.Sku != nil {
			t.Fatalf("storage type %q: want nil Sku on omitted/null sku, got %+v", st.ID, st.Sku)
		}
	}
}

// TestPublicCloudVMStorageTypeListFilter pins the EXACT wire spelling of the two
// paired query params (availabilityZoneId + instanceFamilyId). The spelling can
// only be pinned here and by the live probe: the API silently ignores unknown
// query names (live-probed), so a misspelled filter tag would return the full
// catalogue with HTTP 200 — invisible at runtime. A tag rename turns this RED.
func TestPublicCloudVMStorageTypeListFilter(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/storage_types" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("availabilityZoneId"); got != "az-1" {
			t.Errorf("availabilityZoneId filter not wired: %q", got)
		}
		if got := r.URL.Query().Get("instanceFamilyId"); got != "fam-1" {
			t.Errorf("instanceFamilyId filter not wired: %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storageTypes":[{"id":"st-1","name":"Standard","minSizeGb":1,"maxSizeGb":2048,"isAvailable":true}]}`))
	})
	sts, err := c.PublicCloudVM().StorageType().List(ctx, &PublicCloudVMStorageTypeFilter{
		AvailabilityZoneID: "az-1",
		InstanceFamilyID:   "fam-1",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sts) != 1 || sts[0].ID != "st-1" {
		t.Fatalf("filtered list not decoded: %+v", sts)
	}
}

// TestPublicCloudVMStorageTypeListLoneFilterRefused pins the client-side guard
// for the PAIRED-filters contract: the API answers an opaque HTTP 500 to a
// request carrying exactly one of the two filters (live-probed), so List must
// refuse it BEFORE it reaches the wire, with an actionable error. The handler
// fails the test if the request is ever sent: without the guard (the mutation),
// both sub-tests go RED on "request must never reach the wire".
func TestPublicCloudVMStorageTypeListLoneFilterRefused(t *testing.T) {
	ctx := context.Background()
	for name, filter := range map[string]*PublicCloudVMStorageTypeFilter{
		"availabilityZoneId alone": {AvailabilityZoneID: "az-1"},
		"instanceFamilyId alone":   {InstanceFamilyID: "fam-1"},
	} {
		t.Run(name, func(t *testing.T) {
			c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("request must never reach the wire with a lone filter: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			})
			sts, err := c.PublicCloudVM().StorageType().List(ctx, filter)
			if err == nil {
				t.Fatalf("want an error on a lone filter, got %d storage types", len(sts))
			}
			if !strings.Contains(err.Error(), "must be used together") {
				t.Fatalf("error must state the pairing rule, got: %v", err)
			}
		})
	}
}

func TestPublicCloudVMStorageTypeListEmptyWrapper(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storageTypes":[]}`))
	})
	sts, err := c.PublicCloudVM().StorageType().List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sts) != 0 {
		t.Fatalf("want 0 storage types, got %d", len(sts))
	}
}
