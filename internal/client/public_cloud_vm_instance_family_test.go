package client

import (
	"context"
	"net/http"
	"testing"
)

func TestPublicCloudVMInstanceFamilyList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/instance_families" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"fam-1","name":"Development","description":"Development familly","vcpuMin":1,"vcpuMax":16,"ramMinGb":2,"ramMaxGb":24}]`))
	})
	fams, err := c.PublicCloudVM().InstanceFamily().List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(fams) != 1 {
		t.Fatalf("want 1 family, got %d", len(fams))
	}
	f := fams[0]
	if f.ID != "fam-1" || f.Name != "Development" || f.Description != "Development familly" {
		t.Fatalf("strings not decoded: %+v", f)
	}
	if f.VcpuMin != 1 || f.VcpuMax != 16 || f.RamMinGb != 2 || f.RamMaxGb != 24 {
		t.Fatalf("bounds not decoded: %+v", f)
	}
}

func TestPublicCloudVMInstanceFamilyRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 by id decodes", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/instance_families/fam-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"fam-1","name":"Development","vcpuMin":1,"vcpuMax":16}`))
		})
		f, err := c.PublicCloudVM().InstanceFamily().Read(ctx, "fam-1")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if f == nil || f.ID != "fam-1" || f.VcpuMax != 16 {
			t.Fatalf("bad family: %+v", f)
		}
	})

	t.Run("404 is absence", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
		f, err := c.PublicCloudVM().InstanceFamily().Read(ctx, "missing")
		if err != nil || f != nil {
			t.Fatalf("404 should be (nil,nil), got f=%+v err=%v", f, err)
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		f, err := c.PublicCloudVM().InstanceFamily().Read(ctx, "denied")
		if err == nil || f != nil {
			t.Fatalf("403 must fail closed, got f=%+v err=%v", f, err)
		}
	})
}

// TestPublicCloudVMInstanceFamilySkusDecode pins the exact `skus` API contract
// captured live for #506 (camelCase keys, price as a JSON number that may be
// decimal OR integer, descriptionEn -> DescriptionEn). It fails RED on the code
// that predates the Skus field (it does not even compile), and behaviourally if
// the DescriptionEn json tag or a numeric type ever drifts. The assertions are
// keyed by SKU name so they are robust to the API's element order.
func TestPublicCloudVMInstanceFamilySkusDecode(t *testing.T) {
	ctx := context.Background()

	// One decimal-priced vCPU SKU and one integer-priced RAM SKU: both must
	// decode into float64 (23.42 and 2.0). descriptionEn is the camelCase API key.
	const body = `[{
		"id":"fam-1","name":"General Purpose","vcpuMin":4,"vcpuMax":32,"ramMinGb":4,"ramMaxGb":128,
		"skus":[
			{"name":"csp:fr1:vminstance:gp:vcpu:v1","price":23.42,"unit":"vcpu","description":"Coeur FR","descriptionEn":"Core EN"},
			{"name":"csp:fr1:vminstance:gp:ram:v1","price":2,"unit":"gio","description":"RAM FR","descriptionEn":"RAM EN"}
		]
	}]`

	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
	fams, err := c.PublicCloudVM().InstanceFamily().List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(fams) != 1 {
		t.Fatalf("want 1 family, got %d", len(fams))
	}
	skus := fams[0].Skus
	if len(skus) != 2 {
		t.Fatalf("want 2 skus decoded, got %d (%+v)", len(skus), skus)
	}

	byName := map[string]PublicCloudVMSku{}
	for _, s := range skus {
		byName[s.Name] = s
	}

	vcpu, ok := byName["csp:fr1:vminstance:gp:vcpu:v1"]
	if !ok {
		t.Fatalf("vcpu sku not decoded; got %+v", skus)
	}
	if vcpu.Price != 23.42 {
		t.Errorf("vcpu price = %v, want 23.42 (decimal JSON number)", vcpu.Price)
	}
	if vcpu.Unit != "vcpu" || vcpu.Description != "Coeur FR" || vcpu.DescriptionEn != "Core EN" {
		t.Errorf("vcpu sku fields not decoded: %+v", vcpu)
	}

	ram, ok := byName["csp:fr1:vminstance:gp:ram:v1"]
	if !ok {
		t.Fatalf("ram sku not decoded; got %+v", skus)
	}
	if ram.Price != 2 {
		t.Errorf("ram price = %v, want 2 (integer JSON number decoded as float64)", ram.Price)
	}
	if ram.Unit != "gio" || ram.DescriptionEn != "RAM EN" {
		t.Errorf("ram sku fields not decoded: %+v", ram)
	}
}
