package client

import (
	"context"
	"net/http"
	"testing"
)

// TestPublicCloudVMStorageTypeList pins the WRAPPED-response decode
// ({ "storageTypes": [...] }) including the camelCase int/bool fields.
func TestPublicCloudVMStorageTypeList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/storage_types" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storageTypes":[{"id":"st-1","name":"Standard","description":"Standard block storage","iopsHint":"~1500 IOPS/TB","minSizeGb":1,"maxSizeGb":2048,"isAvailable":true}]}`))
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
