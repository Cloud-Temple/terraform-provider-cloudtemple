package provider

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func siRead(si *client.StaticIP, err error) vpcStaticIPReadFunc {
	return func(ctx context.Context, id string) (*client.StaticIP, error) {
		return si, err
	}
}

func siListStrict(list ...*client.StaticIP) vpcStaticIPListStrictFunc {
	return func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
		return list, nil
	}
}

func siListStrictErr(err error) vpcStaticIPListStrictFunc {
	return func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
		return nil, err
	}
}

// newStaticIPState builds a ResourceData standing for an existing static IP in
// the state, with all the mutable + identity fields seeded.
func newStaticIPState(t *testing.T) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourceVPCStaticIP().Schema, map[string]interface{}{})
	d.SetId("si-1")
	for k, v := range map[string]string{
		"private_network_id":   "pn-1",
		"mac_address":          "00:50:56:ab:cd:ef",
		"ip_address":           "10.0.1.50",
		"resource_description": "seeded",
	} {
		if err := d.Set(k, v); err != nil {
			t.Fatalf("seeding %q: %v", k, err)
		}
	}
	return d
}

// assertStaticIPStatePreserved proves the state entry is left exactly as seeded —
// the core no-auto-drop / no-corruption invariant on an inconclusive read.
func assertStaticIPStatePreserved(t *testing.T, d *schema.ResourceData) {
	t.Helper()
	if d.Id() != "si-1" {
		t.Fatalf("id must be preserved, got %q", d.Id())
	}
	for k, want := range map[string]string{
		"private_network_id":   "pn-1",
		"mac_address":          "00:50:56:ab:cd:ef",
		"ip_address":           "10.0.1.50",
		"resource_description": "seeded",
	} {
		if got := d.Get(k).(string); got != want {
			t.Fatalf("%s must be preserved, got %q (want %q)", k, got, want)
		}
	}
}

// TestReadVPCStaticIPInto pins the read wiring. The overriding invariant: the
// resource is NEVER dropped on an inconclusive read. In readForRefresh, only a
// complete 200 listing that does NOT contain the id drops it; everything else
// fails closed and keeps the state untouched (#275/#281, #303). In readAfterWrite,
// even a confirmed-absent listing fails closed (eventual consistency, never a
// deletion) — the R-N2 mode is what distinguishes the two (#348).
func TestReadVPCStaticIPInto(t *testing.T) {
	ctx := context.Background()

	t.Run("a read error keeps the id and the state", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d, siRead(nil, errors.New("boom")),
			siListStrictErr(errors.New("listing must not be reached on a hard read error")), readForRefresh)
		if !diags.HasError() {
			t.Fatal("a read error must surface as a diagnostic")
		}
		assertStaticIPStatePreserved(t, d)
	})

	t.Run("a read returning a mismatched id fails closed (never rebinds state)", func(t *testing.T) {
		d := newStaticIPState(t)
		// A by-id read that comes back with a DIFFERENT id must not be trusted.
		diags := readVPCStaticIPInto(ctx, d,
			siRead(&client.StaticIP{ID: "someone-else", Source: "custom", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			siListStrictErr(errors.New("listing must not be reached on a mismatched-id read")), readForRefresh)
		if !diags.HasError() {
			t.Fatal("a read returning a different id must fail closed, never rebind the state")
		}
		assertStaticIPStatePreserved(t, d)
	})

	t.Run("a read returning an empty id fails closed (never drops state)", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d,
			siRead(&client.StaticIP{ID: "", Source: "custom", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			siListStrictErr(errors.New("listing must not be reached on an empty-id read")), readForRefresh)
		if !diags.HasError() {
			t.Fatal("a read returning an empty id must fail closed, never write id=\"\" into the state")
		}
		assertStaticIPStatePreserved(t, d)
	})

	t.Run("an inconclusive read with a failing strict listing fails closed", func(t *testing.T) {
		d := newStaticIPState(t)
		// A 206/403/5xx is surfaced by ListStrict as an error -> cannot prove
		// absence -> must keep the resource.
		diags := readVPCStaticIPInto(ctx, d, siRead(nil, nil), siListStrictErr(errors.New("206 partial")), readForRefresh)
		if !diags.HasError() {
			t.Fatal("a failing strict listing must fail closed with a diagnostic, never a drop")
		}
		assertStaticIPStatePreserved(t, d)
	})

	t.Run("an inconclusive read but still listed fails closed (never drops)", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d, siRead(nil, nil),
			siListStrict(&client.StaticIP{ID: "other"}, &client.StaticIP{ID: "si-1"}), readForRefresh)
		if !diags.HasError() {
			t.Fatal("a still-listed static IP must fail closed, never auto-remove")
		}
		assertStaticIPStatePreserved(t, d)
	})

	t.Run("an inconclusive read with a missing private_network_id fails closed", func(t *testing.T) {
		d := newStaticIPState(t)
		if err := d.Set("private_network_id", ""); err != nil {
			t.Fatalf("clearing private_network_id: %v", err)
		}
		diags := readVPCStaticIPInto(ctx, d, siRead(nil, nil),
			siListStrictErr(errors.New("listing must not be reached without a scope")), readForRefresh)
		if !diags.HasError() {
			t.Fatal("a missing private_network_id must fail closed, never auto-remove")
		}
		if d.Id() != "si-1" {
			t.Fatalf("id must be preserved, got %q", d.Id())
		}
	})

	t.Run("readForRefresh: a confirmed-absent strict listing DROPS the resource", func(t *testing.T) {
		d := newStaticIPState(t)
		// Complete 200 listing of the private network that does NOT contain si-1:
		// genuine deletion evidence -> drop (R-Q10).
		diags := readVPCStaticIPInto(ctx, d, siRead(nil, nil),
			siListStrict(&client.StaticIP{ID: "other"}, &client.StaticIP{ID: "yet-another"}), readForRefresh)
		if diags.HasError() {
			t.Fatalf("a confirmed absence must drop cleanly, got: %v", diags)
		}
		if d.Id() != "" {
			t.Fatalf("a confirmed-absent static IP must be dropped (SetId(\"\")), got id %q", d.Id())
		}
	})

	t.Run("readForRefresh: an empty 200 listing DROPS the resource", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d, siRead(nil, nil), siListStrict(), readForRefresh)
		if diags.HasError() {
			t.Fatalf("an empty complete listing is a confirmed absence, got: %v", diags)
		}
		if d.Id() != "" {
			t.Fatalf("an empty 200 listing must drop the resource, got id %q", d.Id())
		}
	})

	t.Run("readForRefresh: substring/superstring ids do not false-match the listing", func(t *testing.T) {
		d := newStaticIPState(t)
		// None of these equal "si-1" exactly -> still a confirmed absence -> drop.
		diags := readVPCStaticIPInto(ctx, d, siRead(nil, nil),
			siListStrict(&client.StaticIP{ID: "si-12"}, &client.StaticIP{ID: "si-"}, &client.StaticIP{ID: "si"}), readForRefresh)
		if diags.HasError() {
			t.Fatalf("only an exact id match is liveness; near-misses are an absence, got: %v", diags)
		}
		if d.Id() != "" {
			t.Fatalf("near-miss ids must not keep the resource alive, got id %q", d.Id())
		}
	})

	// R-N2 / #348: the SAME confirmed-absent input that DROPS in readForRefresh must
	// FAIL CLOSED (keep the id) in readAfterWrite — right after a write an absent
	// listing is eventual consistency, not a deletion. This kills the mutant that
	// would reuse readForRefresh on the create/update path and orphan a fresh id.
	t.Run("readAfterWrite: a confirmed-absent strict listing FAILS CLOSED and keeps the id", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d, siRead(nil, nil),
			siListStrict(&client.StaticIP{ID: "other"}, &client.StaticIP{ID: "yet-another"}), readAfterWrite)
		if !diags.HasError() {
			t.Fatal("readAfterWrite must FAIL CLOSED on a confirmed-absent listing (eventual consistency), never drop a just-written id")
		}
		if d.Id() != "si-1" {
			t.Fatalf("readAfterWrite must keep the just-written id, got %q (a drop here would orphan the static IP)", d.Id())
		}
	})

	t.Run("readAfterWrite: an empty 200 listing FAILS CLOSED and keeps the id", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d, siRead(nil, nil), siListStrict(), readAfterWrite)
		if !diags.HasError() {
			t.Fatal("readAfterWrite must FAIL CLOSED on an empty listing, never drop a just-written id")
		}
		if d.Id() != "si-1" {
			t.Fatalf("readAfterWrite must keep the just-written id, got %q", d.Id())
		}
	})

	t.Run("a successful read repopulates the full flatten keyset", func(t *testing.T) {
		d := newStaticIPState(t)
		desc := "fresh"
		si := &client.StaticIP{
			ID:                  "si-1",
			IPAddress:           "10.0.1.99",
			MacAddress:          "00:50:56:aa:bb:cc",
			Source:              "custom",
			ResourceDescription: &desc,
			VirtualMachine:      &client.BaseObject{ID: "vm-1"},
			NetworkAdapter:      &client.BaseObject{ID: "na-1"},
			FloatingIP:          &client.StaticIPFloatingIP{ID: "fip-1", IPAddress: "198.51.100.7"},
			VPC:                 client.BaseObject{ID: "vpc-1"},
			PrivateNetwork:      client.BaseObject{ID: "pn-1"},
		}
		diags := readVPCStaticIPInto(ctx, d, siRead(si, nil),
			siListStrictErr(errors.New("listing must not be reached on a successful read")), readForRefresh)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		want := map[string]string{
			"ip_address":           "10.0.1.99",
			"source":               "custom",
			"resource_description": "fresh",
			"virtual_machine_id":   "vm-1",
			"network_adapter_id":   "na-1",
			"floating_ip_id":       "fip-1",
			"floating_ip_address":  "198.51.100.7",
			"vpc_id":               "vpc-1",
			"private_network_id":   "pn-1",
		}
		for k, v := range want {
			if got := d.Get(k).(string); got != v {
				t.Fatalf("%s = %q, want %q", k, got, v)
			}
		}
	})

	t.Run("nil VirtualMachine/NetworkAdapter/FloatingIP flatten to empty strings without panic", func(t *testing.T) {
		d := newStaticIPState(t)
		si := &client.StaticIP{
			ID:             "si-1",
			IPAddress:      "10.0.1.99",
			MacAddress:     "00:50:56:aa:bb:cc",
			Source:         "custom", // a TF-managed static IP is always custom (#311 guard)
			VPC:            client.BaseObject{ID: "vpc-1"},
			PrivateNetwork: client.BaseObject{ID: "pn-1"},
			// VirtualMachine, NetworkAdapter, ResourceDescription, FloatingIP all nil.
		}
		diags := readVPCStaticIPInto(ctx, d, siRead(si, nil), siListStrict(), readForRefresh)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		for _, k := range []string{"virtual_machine_id", "network_adapter_id", "floating_ip_id", "floating_ip_address", "resource_description"} {
			if got := d.Get(k).(string); got != "" {
				t.Fatalf("nil association %s must flatten to \"\", got %q", k, got)
			}
		}
	})
}

// TestVPCStaticIPResourceSchemaKeyset pins that the resource schema declares
// EXACTLY the FlattenStaticIP keyset plus nothing else. A swap/drop of a flat
// field (or a stray schema key) goes RED here.
func TestVPCStaticIPResourceSchemaKeyset(t *testing.T) {
	flat := map[string]bool{
		"id": true, "ip_address": true, "mac_address": true, "source": true,
		"vpc_id": true, "private_network_id": true, "virtual_machine_id": true,
		"network_adapter_id": true, "resource_description": true,
		"floating_ip_id": true, "floating_ip_address": true,
	}

	schemaKeys := map[string]bool{}
	for k := range resourceVPCStaticIP().Schema {
		schemaKeys[k] = true
	}

	if len(schemaKeys) != len(flat) {
		var sk, fk []string
		for k := range schemaKeys {
			sk = append(sk, k)
		}
		for k := range flat {
			fk = append(fk, k)
		}
		sort.Strings(sk)
		sort.Strings(fk)
		t.Fatalf("schema keyset must equal the FlattenStaticIP keyset.\nschema:  %v\nflatten: %v", sk, fk)
	}
	for k := range flat {
		if !schemaKeys[k] {
			t.Fatalf("FlattenStaticIP key %q is missing from the resource schema", k)
		}
	}
	for k := range schemaKeys {
		if !flat[k] {
			t.Fatalf("resource schema has key %q that FlattenStaticIP does not emit", k)
		}
	}
}

// TestVPCStaticIPResourceDescriptionSchema pins R-B1: resource_description is
// Optional with a non-empty Default and a ValidateFunc that rejects an empty or
// whitespace-only value at plan time (the live API requires a non-empty
// resourceDescription). Non-complacent: dropping the Default or the ValidateFunc,
// or accepting an empty value, goes RED here.
func TestVPCStaticIPResourceDescriptionSchema(t *testing.T) {
	s, ok := resourceVPCStaticIP().Schema["resource_description"]
	if !ok {
		t.Fatal("resource_description attribute is missing from the schema")
	}
	if s.Default != "Managed by Terraform" {
		t.Fatalf("resource_description Default = %v, want \"Managed by Terraform\" (R-B1)", s.Default)
	}
	if s.ValidateFunc == nil {
		t.Fatal("resource_description must carry a ValidateFunc rejecting empty/whitespace (R-B1)")
	}
	for _, bad := range []string{"", "   ", "\t"} {
		if _, errs := s.ValidateFunc(bad, "resource_description"); len(errs) == 0 {
			t.Fatalf("resource_description must reject %q (empty/whitespace), got no error", bad)
		}
	}
	for _, good := range []string{"Managed by Terraform", "prod db static IP"} {
		if _, errs := s.ValidateFunc(good, "resource_description"); len(errs) != 0 {
			t.Fatalf("resource_description must accept %q, got %v", good, errs)
		}
	}
}

// TestVPCStaticIPMACStateFunc pins the MAC canonicalisation StateFunc: the schema
// stores the lowercase ":"-separated form regardless of the (regexp-accepted)
// input casing/separator, so a config written in an equivalent form does not show
// a perpetual no-op plan against the API's canonical read-back. Non-complacent:
// removing the StateFunc (nil) or one that fails to canonicalise goes RED here.
func TestVPCStaticIPMACStateFunc(t *testing.T) {
	s, ok := resourceVPCStaticIP().Schema["mac_address"]
	if !ok {
		t.Fatal("mac_address attribute is missing from the schema")
	}
	if s.StateFunc == nil {
		t.Fatal("mac_address must carry a StateFunc canonicalising the MAC (else an uppercase/dash config shows a perpetual no-op plan)")
	}
	cases := map[string]string{
		"00-50-56-AB-CD-EF": "00:50:56:ab:cd:ef", // dashes + uppercase
		"AA:BB:CC:DD:EE:FF": "aa:bb:cc:dd:ee:ff", // uppercase only
		"00-50:56-AB:cd-EF": "00:50:56:ab:cd:ef", // mixed separators + mixed case (the regexp admits each [:-] independently)
		"00:50:56:ab:cd:ef": "00:50:56:ab:cd:ef", // already canonical -> unchanged (idempotent)
	}
	for in, want := range cases {
		if got := s.StateFunc(in); got != want {
			t.Fatalf("StateFunc(%q) = %q, want %q", in, got, want)
		}
	}
}
