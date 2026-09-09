package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/golang-jwt/jwt/v4"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// newStorageTypeTestClient wires a Client to a stub HTTP server with a
// pre-seeded far-future JWT, so JWT() never hits the network: these tests
// exercise the paired-filter validation, not the auth plumbing.
func newStorageTypeTestClient(t *testing.T, h http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := client.NewClient(&client.Config{Address: srv.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SavedToken = &jwt.Token{
		Claims: jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())},
	}
	return c
}

const (
	stTestAZ       = "11111111-1111-1111-1111-111111111111"
	stTestFamily   = "22222222-2222-2222-2222-222222222222"
	stTestOtherFam = "33333333-3333-3333-3333-333333333333"
)

// storageTypeStubHandler serves the three catalogue endpoints the filtered read
// touches. Each boolean/knob shapes one failure scenario; catalogueMustNotBeHit
// turns any /storage_types request into a test failure — the fail-closed
// ordering assertion (validation FIRST, catalogue only if the pair is proven).
type storageTypeStubKnobs struct {
	azExists              bool
	azCompatibleFamilies  string // JSON array; "" omits the field entirely
	familyExists          bool
	catalogueMustNotBeHit bool
}

func storageTypeStubHandler(t *testing.T, k storageTypeStubKnobs) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/vm_instances/v1/availability_zones/"+stTestAZ:
			if !k.azExists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			body := `{"id":"` + stTestAZ + `","name":"fr1-az01","isEnabled":true`
			if k.azCompatibleFamilies != "" {
				body += `,"compatibleFamilies":` + k.azCompatibleFamilies
			}
			body += `}`
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		case r.URL.Path == "/vm_instances/v1/instance_families/"+stTestFamily:
			if !k.familyExists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"` + stTestFamily + `","name":"Development","ramMinGib":1,"ramMaxGib":64}`))
		case r.URL.Path == "/vm_instances/v1/storage_types":
			if k.catalogueMustNotBeHit {
				t.Errorf("storage_types must not be queried when the filter pair fails validation (fail-closed ordering)")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"storageTypes":[{"id":"27bdd5f3-43f8-4c36-b344-d7c5452f8eb2","name":"Enterprise","minSizeGb":1,"maxSizeGb":2048,"isAvailable":true}]}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

// TestStorageTypesFilterPairValidation pins the fail-closed guard against the
// API's silently-ignored filter VALUES (live-probed: a nonexistent id returns
// the FULL catalogue with HTTP 200): a nonexistent zone or family, or a family
// the zone does not advertise as compatible, must error BEFORE the catalogue is
// queried — never silently return an unfiltered list. Mutation: with the
// validate call removed from the read, every failing case here goes RED (the
// read succeeds and /storage_types is hit).
func TestStorageTypesFilterPairValidation(t *testing.T) {
	ctx := context.Background()
	compat := `[{"id":"` + stTestFamily + `","name":"Development"}]`
	otherCompat := `[{"id":"` + stTestOtherFam + `","name":"Other"}]`

	cases := []struct {
		name    string
		knobs   storageTypeStubKnobs
		wantErr string // "" means the read must succeed
	}{
		{
			name:    "nonexistent availability zone fails closed",
			knobs:   storageTypeStubKnobs{azExists: false, familyExists: true, catalogueMustNotBeHit: true},
			wantErr: "does not exist",
		},
		{
			name:    "nonexistent instance family fails closed",
			knobs:   storageTypeStubKnobs{azExists: true, azCompatibleFamilies: compat, familyExists: false, catalogueMustNotBeHit: true},
			wantErr: "does not exist",
		},
		{
			name:    "family not advertised as compatible with the zone fails closed",
			knobs:   storageTypeStubKnobs{azExists: true, azCompatibleFamilies: otherCompat, familyExists: true, catalogueMustNotBeHit: true},
			wantErr: "not advertised as compatible",
		},
		{
			name:  "valid and compatible pair reads the catalogue",
			knobs: storageTypeStubKnobs{azExists: true, azCompatibleFamilies: compat, familyExists: true},
		},
		{
			name: "zone not advertising compatible families skips the compatibility check",
			// An empty/absent compatibleFamilies cannot prove an incompatibility:
			// the pair must be accepted (no false fail-closed against an API that
			// does not populate the relationship).
			knobs: storageTypeStubKnobs{azExists: true, azCompatibleFamilies: "", familyExists: true},
		},
	}

	for _, tc := range cases {
		t.Run("plural/"+tc.name, func(t *testing.T) {
			c := newStorageTypeTestClient(t, storageTypeStubHandler(t, tc.knobs))
			d := schema.TestResourceDataRaw(t, dataSourcePublicCloudVMStorageTypes().Schema, map[string]interface{}{
				"availability_zone_id": stTestAZ,
				"instance_family_id":   stTestFamily,
			})
			diags := publicCloudVMStorageTypesRead(ctx, d, c)
			if tc.wantErr == "" {
				if diags.HasError() {
					t.Fatalf("want success, got diags: %+v", diags)
				}
				if n := len(d.Get("storage_types").([]interface{})); n != 1 {
					t.Fatalf("want 1 storage type in state, got %d", n)
				}
				return
			}
			if !diags.HasError() {
				t.Fatalf("want an error containing %q, got success", tc.wantErr)
			}
			if !strings.Contains(diags[0].Summary, tc.wantErr) {
				t.Fatalf("want error containing %q, got: %s", tc.wantErr, diags[0].Summary)
			}
		})
		t.Run("singular/"+tc.name, func(t *testing.T) {
			c := newStorageTypeTestClient(t, storageTypeStubHandler(t, tc.knobs))
			d := schema.TestResourceDataRaw(t, dataSourcePublicCloudVMStorageType().Schema, map[string]interface{}{
				"name":                 "Enterprise",
				"availability_zone_id": stTestAZ,
				"instance_family_id":   stTestFamily,
			})
			diags := publicCloudVMStorageTypeRead(ctx, d, c)
			if tc.wantErr == "" {
				if diags.HasError() {
					t.Fatalf("want success, got diags: %+v", diags)
				}
				if got := d.Id(); got != "27bdd5f3-43f8-4c36-b344-d7c5452f8eb2" {
					t.Fatalf("want the matched storage type id, got %q", got)
				}
				return
			}
			if !diags.HasError() {
				t.Fatalf("want an error containing %q, got success", tc.wantErr)
			}
			if !strings.Contains(diags[0].Summary, tc.wantErr) {
				t.Fatalf("want error containing %q, got: %s", tc.wantErr, diags[0].Summary)
			}
		})
	}
}

// TestStorageTypesUnfilteredSkipsValidation pins that an UNFILTERED read stays a
// single catalogue GET: the zone/family endpoints must not be queried (the
// validation is scoped to the paired filters, not a new tax on every read).
func TestStorageTypesUnfilteredSkipsValidation(t *testing.T) {
	ctx := context.Background()
	c := newStorageTypeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vm_instances/v1/storage_types" {
			t.Errorf("unfiltered read must only query the catalogue, got: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storageTypes":[{"id":"27bdd5f3-43f8-4c36-b344-d7c5452f8eb2","name":"Enterprise","minSizeGb":1,"maxSizeGb":2048,"isAvailable":true}]}`))
	})
	d := schema.TestResourceDataRaw(t, dataSourcePublicCloudVMStorageTypes().Schema, map[string]interface{}{})
	if diags := publicCloudVMStorageTypesRead(ctx, d, c); diags.HasError() {
		t.Fatalf("unfiltered read failed: %+v", diags)
	}
}

// TestStorageTypeNotFoundErrorNamesTheFilters pins that a by-id miss inside a
// narrowed catalogue names the filter pair, so it is not mistaken for a global
// absence of the storage type.
func TestStorageTypeNotFoundErrorNamesTheFilters(t *testing.T) {
	ctx := context.Background()
	compat := `[{"id":"` + stTestFamily + `","name":"Development"}]`
	c := newStorageTypeTestClient(t, storageTypeStubHandler(t, storageTypeStubKnobs{
		azExists: true, azCompatibleFamilies: compat, familyExists: true,
	}))
	d := schema.TestResourceDataRaw(t, dataSourcePublicCloudVMStorageType().Schema, map[string]interface{}{
		"id":                   "44444444-4444-4444-4444-444444444444",
		"availability_zone_id": stTestAZ,
		"instance_family_id":   stTestFamily,
	})
	diags := publicCloudVMStorageTypeRead(ctx, d, c)
	if !diags.HasError() {
		t.Fatalf("want a not-found error, got success")
	}
	if !strings.Contains(diags[0].Summary, "within the catalogue filtered by") ||
		!strings.Contains(diags[0].Summary, stTestAZ) || !strings.Contains(diags[0].Summary, stTestFamily) {
		t.Fatalf("not-found error must name the filter pair, got: %s", diags[0].Summary)
	}
}
