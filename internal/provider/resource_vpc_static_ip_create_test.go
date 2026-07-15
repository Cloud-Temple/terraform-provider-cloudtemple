package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// newStaticIPCreateData builds a ResourceData for a fresh create: the create
// inputs are seeded but NO id is set yet (the id is resolved from the async
// create activity by the client and set by the provider).
func newStaticIPCreateData(t *testing.T) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourceVPCStaticIP().Schema, map[string]interface{}{})
	for k, v := range map[string]string{
		"private_network_id":   "pn-1",
		"mac_address":          "00:50:56:ab:cd:ef",
		"ip_address":           "10.0.1.50",
		"resource_description": "managed",
	} {
		if err := d.Set(k, v); err != nil {
			t.Fatalf("seeding %q: %v", k, err)
		}
	}
	return d
}

// readFatal / listFatal are "must not be reached" sentinels: they fail the test
// if the create orchestration calls them on a path where it must not.
func readFatal(t *testing.T) vpcStaticIPReadFunc {
	return func(ctx context.Context, id string) (*client.StaticIP, error) {
		t.Fatal("read must not be reached when create fails or returns no id")
		return nil, nil
	}
}
func listFatal(t *testing.T) vpcStaticIPListStrictFunc {
	return func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
		t.Fatal("strict listing must not be reached when the per-id read succeeds")
		return nil, nil
	}
}

// TestCreateVPCStaticIPWith pins the async create orchestration (R-Q2, #348). The
// worst outcome is an ORPHAN (created platform-side, absent from the state), so
// state safety drives every failure mode:
//
//   - a create error -> FAIL, never SetId (nothing was confirmed to track);
//   - the client returning an empty id with no error -> FAIL CLOSED (defence in
//     depth against an R-Q2 contract regression that would otherwise SetId(""));
//   - a successful create -> SetId, then a read in readAfterWrite mode that NEVER
//     drops the just-created id on an eventually-consistent listing (#348).
func TestCreateVPCStaticIPWith(t *testing.T) {
	ctx := context.Background()

	t.Run("a create error fails and never sets the id", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		funcs := vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, privateNetworkID string, req *client.CreateStaticIPRequest) (string, error) {
				return "", errors.New("boom")
			},
			read:       readFatal(t),
			listStrict: listFatal(t),
		}
		diags := createVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("a create error must surface as a diagnostic")
		}
		if d.Id() != "" {
			t.Fatalf("a create error must NEVER SetId, got %q", d.Id())
		}
	})

	// R-Q2 guard in depth: the client must never return ("", nil), but if a contract
	// regression did, the provider must FAIL CLOSED rather than SetId("") and silently
	// orphan the static IP. A mutant that trusted the empty id reds here.
	t.Run("an empty id with no error fails closed and never sets the id", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		funcs := vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, privateNetworkID string, req *client.CreateStaticIPRequest) (string, error) {
				return "", nil
			},
			read:       readFatal(t),
			listStrict: listFatal(t),
		}
		diags := createVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("an empty id with no error must FAIL CLOSED (R-Q2 no-id fatal)")
		}
		if d.Id() != "" {
			t.Fatalf("a no-id create must NEVER SetId, got %q", d.Id())
		}
	})

	t.Run("a successful create sets the id, sends the right body, and flattens the read-back", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		var gotPN string
		var gotReq *client.CreateStaticIPRequest
		si := &client.StaticIP{
			ID:                  "si-99",
			IPAddress:           "10.0.1.50",
			MacAddress:          "00:50:56:ab:cd:ef",
			Source:              "custom",
			ResourceDescription: siDescPtr("managed"),
			VPC:                 client.BaseObject{ID: "vpc-1"},
			PrivateNetwork:      client.BaseObject{ID: "pn-1"},
		}
		funcs := vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, privateNetworkID string, req *client.CreateStaticIPRequest) (string, error) {
				gotPN = privateNetworkID
				gotReq = req
				return "si-99", nil
			},
			read:       siRead(si, nil),
			listStrict: listFatal(t), // a successful per-id read must not reach the listing
		}
		diags := createVPCStaticIPWith(ctx, d, funcs)
		if diags.HasError() {
			t.Fatalf("a successful create must succeed, got: %v", diags)
		}
		if d.Id() != "si-99" {
			t.Fatalf("the created id must be set, got %q", d.Id())
		}
		// The create body is built from the ResourceData, scoped to its private network.
		if gotPN != "pn-1" {
			t.Fatalf("create must target private network %q, got %q", "pn-1", gotPN)
		}
		if gotReq.MacAddress != "00:50:56:ab:cd:ef" {
			t.Fatalf("create body MacAddress = %q, want the configured MAC", gotReq.MacAddress)
		}
		if gotReq.IPAddress != "10.0.1.50" {
			t.Fatalf("create body IPAddress = %q, want the configured address", gotReq.IPAddress)
		}
		if gotReq.ResourceDescription != "managed" {
			t.Fatalf("create body ResourceDescription = %q, want the configured description", gotReq.ResourceDescription)
		}
		// The read-back is flattened into the computed attributes.
		for k, want := range map[string]string{
			"source": "custom", "vpc_id": "vpc-1", "ip_address": "10.0.1.50",
			"resource_description": "managed",
		} {
			if got := d.Get(k).(string); got != want {
				t.Fatalf("post-create %s = %q, want %q", k, got, want)
			}
		}
	})

	// ⭐ #348 / R-N2 never-orphan: a just-created static IP that is not YET visible in
	// the strict listing (eventual consistency) must NOT be dropped. The create path
	// uses readAfterWrite, so the SAME confirmed-absent listing that DROPS on refresh
	// here FAILS CLOSED keeping the id. A mutant that passed readForRefresh on the
	// create path would SetId("") and orphan the static IP -> reds here on BOTH checks.
	t.Run("create OK + read nil + confirmed-absent listing keeps the id (never orphan)", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		funcs := vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, privateNetworkID string, req *client.CreateStaticIPRequest) (string, error) {
				return "si-99", nil
			},
			read:       siRead(nil, nil),                            // not yet readable by id
			listStrict: siListStrict(&client.StaticIP{ID: "other"}), // si-99 absent (eventual consistency)
		}
		diags := createVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("a just-created static IP absent from the listing must FAIL CLOSED (eventual consistency), not report a clean apply")
		}
		if d.Id() != "si-99" {
			t.Fatalf("the just-created id must be KEPT (never orphaned), got %q", d.Id())
		}
	})

	t.Run("create OK + read nil + still-listed keeps the id and fails closed", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		funcs := vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, privateNetworkID string, req *client.CreateStaticIPRequest) (string, error) {
				return "si-99", nil
			},
			read:       siRead(nil, nil),
			listStrict: siListStrict(&client.StaticIP{ID: "si-99"}), // present but not yet readable by id
		}
		diags := createVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("a created-but-not-yet-readable static IP must fail closed (attributes could not be populated)")
		}
		if d.Id() != "si-99" {
			t.Fatalf("the just-created id must be kept, got %q", d.Id())
		}
	})

	// Defence in depth: a POST-created static IP is always custom, but if the read-back
	// ever reported a non-custom source the #311 guard must reject it rather than adopt
	// an undeletable resource. The id is kept (it exists platform-side) for the operator.
	t.Run("create OK + non-custom read-back is rejected by the source guard", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		funcs := vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, privateNetworkID string, req *client.CreateStaticIPRequest) (string, error) {
				return "si-99", nil
			},
			read:       siRead(&client.StaticIP{ID: "si-99", Source: "xoa", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			listStrict: listFatal(t),
		}
		diags := createVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("a non-custom read-back must be rejected by the #311 source guard")
		}
		if d.Id() != "si-99" {
			t.Fatalf("the id must be kept on a source-guard rejection (it exists platform-side), got %q", d.Id())
		}
	})

	// Companion to the xoa case: an EMPTY source on the create read-back is NOT
	// positive proof of a custom static IP (#311), so it must FAIL CLOSED — and
	// because the create path reads in readAfterWrite mode, the just-created id is
	// KEPT (never orphaned), never dropped. Reds if the read guard tolerates an
	// empty source again (the fail-OPEN regression).
	t.Run("create OK + empty-source read-back fails closed and keeps the id", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		funcs := vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, privateNetworkID string, req *client.CreateStaticIPRequest) (string, error) {
				return "si-99", nil
			},
			read:       siRead(&client.StaticIP{ID: "si-99", Source: "", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			listStrict: listFatal(t),
		}
		diags := createVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("an empty-source read-back is not proof of a custom static IP; create must FAIL CLOSED, not report a clean apply")
		}
		if d.Id() != "si-99" {
			t.Fatalf("the just-created id must be KEPT on a source-guard rejection (it exists platform-side), got %q", d.Id())
		}
	})
}
