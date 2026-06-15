package provider

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestImportVPCFloatingIPBinding pins the import contract: parse strictly into
// two non-empty halves, then CONFIRM the pair exists by a read. A phantom import
// (FIP not bound to this static IP, or absent) must fail rather than enter the
// state.
func TestImportVPCFloatingIPBinding(t *testing.T) {
	ctx := context.Background()

	newImportData := func(t *testing.T, id string) *schema.ResourceData {
		t.Helper()
		d := schema.TestResourceDataRaw(t, resourceVPCFloatingIPBinding().Schema, map[string]interface{}{})
		d.SetId(id)
		return d
	}

	t.Run("a good id confirmed by the read imports and seeds in/out attrs", func(t *testing.T) {
		d := newImportData(t, testBindID)
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return boundFIP(), nil },
		}
		got, err := importVPCFloatingIPBinding(ctx, d, funcs)
		if err != nil {
			t.Fatalf("a good confirmed import must succeed, got: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("import must return exactly one ResourceData, got %d", len(got))
		}
		if d.Get("floating_ip_id").(string) != testFIPID || d.Get("static_ip_id").(string) != testStaticID {
			t.Fatalf("import must seed the in attrs, got fip=%q static=%q", d.Get("floating_ip_id"), d.Get("static_ip_id"))
		}
		if d.Get("static_ip_address").(string) != "10.0.1.5" {
			t.Fatalf("import must seed the computed attrs, got %q", d.Get("static_ip_address"))
		}
	})

	t.Run("an empty id is rejected", func(t *testing.T) {
		d := newImportData(t, "")
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
				return nil, errors.New("read must not be reached on a malformed id")
			},
		}
		if _, err := importVPCFloatingIPBinding(ctx, d, funcs); err == nil {
			t.Fatal("an empty composite id must be rejected before any read")
		}
	})

	t.Run("a malformed id (one part) is rejected", func(t *testing.T) {
		d := newImportData(t, "fip-only")
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
				return nil, errors.New("read must not be reached on a malformed id")
			},
		}
		if _, err := importVPCFloatingIPBinding(ctx, d, funcs); err == nil {
			t.Fatal("a one-part composite id must be rejected before any read")
		}
	})

	t.Run("a well-formed id whose FIP is NOT bound to the static IP is a phantom import (rejected)", func(t *testing.T) {
		d := newImportData(t, testBindID)
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return otherBoundFIP(), nil },
		}
		if _, err := importVPCFloatingIPBinding(ctx, d, funcs); err == nil {
			t.Fatal("importing a FIP not bound to the requested static IP must fail (no phantom import)")
		}
	})

	t.Run("a well-formed id whose FIP is unbound is rejected", func(t *testing.T) {
		d := newImportData(t, testBindID)
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return unboundFIP(), nil },
		}
		if _, err := importVPCFloatingIPBinding(ctx, d, funcs); err == nil {
			t.Fatal("importing an unbound FIP must fail (the pair does not exist)")
		}
	})

	t.Run("a well-formed id whose FIP read is ambiguous (nil) is rejected", func(t *testing.T) {
		d := newImportData(t, testBindID)
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, nil },
		}
		if _, err := importVPCFloatingIPBinding(ctx, d, funcs); err == nil {
			t.Fatal("importing a FIP that cannot be read must fail (do not import a phantom)")
		}
	})

	t.Run("a read error during import is surfaced", func(t *testing.T) {
		d := newImportData(t, testBindID)
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, errors.New("boom") },
		}
		if _, err := importVPCFloatingIPBinding(ctx, d, funcs); err == nil {
			t.Fatal("a read error during import must be surfaced")
		}
	})
}

// TestVPCFloatingIPBindingSchemaKeyset pins the resource schema's exact keyset.
// A stray or dropped attribute (e.g. a `description` slipping in, which the
// resource must NEVER manage) goes RED here.
func TestVPCFloatingIPBindingSchemaKeyset(t *testing.T) {
	want := map[string]bool{
		"floating_ip_id":      true,
		"static_ip_id":        true,
		"floating_ip_address": true,
		"static_ip_address":   true,
	}
	got := map[string]bool{}
	for k := range resourceVPCFloatingIPBinding().Schema {
		got[k] = true
	}
	if len(got) != len(want) {
		var gk, wk []string
		for k := range got {
			gk = append(gk, k)
		}
		for k := range want {
			wk = append(wk, k)
		}
		sort.Strings(gk)
		sort.Strings(wk)
		t.Fatalf("schema keyset mismatch.\ngot:  %v\nwant: %v", gk, wk)
	}
	for k := range want {
		if !got[k] {
			t.Fatalf("missing schema key %q", k)
		}
	}
	// The resource must NOT manage the floating IP description.
	if got["description"] {
		t.Fatal("the binding resource must NOT declare a `description` attribute (it does not manage the FIP description)")
	}
}

// TestVPCFloatingIPBindingNoUpdate pins that both attributes are ForceNew and
// the resource declares NO Update (create = bind, delete = unbind).
func TestVPCFloatingIPBindingNoUpdate(t *testing.T) {
	r := resourceVPCFloatingIPBinding()
	if r.UpdateContext != nil {
		t.Fatal("the binding resource must NOT declare an Update (both attributes are ForceNew)")
	}
	for _, k := range []string{"floating_ip_id", "static_ip_id"} {
		if !r.Schema[k].ForceNew {
			t.Fatalf("%q must be ForceNew", k)
		}
		if !r.Schema[k].Required {
			t.Fatalf("%q must be Required", k)
		}
	}
	for _, k := range []string{"floating_ip_address", "static_ip_address"} {
		if !r.Schema[k].Computed {
			t.Fatalf("%q must be Computed (read-only)", k)
		}
	}
}
