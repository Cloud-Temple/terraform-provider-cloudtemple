package client

import (
	"context"
	"math"
	"net/http"
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
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storageTypes":[{"id":"st-1","name":"Standard","description":"Standard block storage","iopsHint":"~1500 IOPS/TB","minSizeGb":1,"maxSizeGb":2048,"isAvailable":true,"sku":{"name":"csp:fr1:iaas:storage:bloc:medium:v1","price":0.063,"unit":"1 Gio","description":"Stockage bloc medium","descriptionEn":"Medium block storage"}}]}`))
	})
	sts, err := c.PublicCloudVM().StorageType().List(ctx)
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
	sts, err := c.PublicCloudVM().StorageType().List(ctx)
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

func TestPublicCloudVMStorageTypeListEmptyWrapper(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storageTypes":[]}`))
	})
	sts, err := c.PublicCloudVM().StorageType().List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sts) != 0 {
		t.Fatalf("want 0 storage types, got %d", len(sts))
	}
}
