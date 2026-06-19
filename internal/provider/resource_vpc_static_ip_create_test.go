package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// newStaticIPCreateData builds a ResourceData standing for a static IP being
// CREATED: the input fields are seeded but NO id is set yet (the id is learned
// from the completed create activity, never from the request body).
func newStaticIPCreateData(t *testing.T) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourceVPCStaticIP().Schema, map[string]interface{}{})
	for k, v := range map[string]string{
		"private_network_id":   "pn-1",
		"mac_address":          "00:50:56:ab:cd:ef",
		"ip_address":           "10.0.1.50",
		"resource_description": "seeded-desc",
	} {
		if err := d.Set(k, v); err != nil {
			t.Fatalf("seeding %q: %v", k, err)
		}
	}
	return d
}

// completedActivity mirrors the shape setIdFromActivityState consumes: a single
// state entry whose Result carries the created static IP id.
func completedActivity(result string) *client.Activity {
	return &client.Activity{State: map[string]client.ActivityState{"step": {Result: result}}}
}

// TestCreateVPCStaticIPWith pins the ASYNCHRONOUS create contract end to end
// (#348). Create returns an activity (Location), and ONLY the COMPLETED activity
// carries the new static IP id (in its state Result) — never the response body.
// State safety drives every failure mode; the worst outcome is an ORPHAN (created
// platform-side, absent from the state). Each subtest is mutation-proven:
//   - drop the activity-id threading and the request/threading subtest reds;
//   - drop the wait-error fail-closed and its "import" diagnostic reds;
//   - drop the empty-Result guard and the read is wrongly reached / no error;
//   - drop the create-mode (dropOnConfirmedAbsence=false) wiring and the
//     eventually-consistent-absence subtest orphans the id (reds).
func TestCreateVPCStaticIPWith(t *testing.T) {
	ctx := context.Background()

	noWait := func(ctx context.Context, activityID string) (*client.Activity, error) {
		return nil, errors.New("wait must not be reached")
	}
	noRead := func(ctx context.Context, id string) (*client.StaticIP, error) {
		return nil, errors.New("read must not be reached")
	}
	noList := func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
		return nil, errors.New("listing must not be reached")
	}

	t.Run("a create error fails and sets no id", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		diags := createVPCStaticIPWith(ctx, d, vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, pn string, req *client.CreateStaticIPRequest) (string, error) {
				return "", errors.New("boom")
			},
			wait: noWait, read: noRead, listStrict: noList,
		})
		if !diags.HasError() {
			t.Fatal("a create error must surface as a diagnostic")
		}
		if d.Id() != "" {
			t.Fatalf("a failed create must leave the id empty (nothing was confirmed created), got %q", d.Id())
		}
	})

	t.Run("the create request carries the configured fields and its activity id is awaited", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		var gotReq *client.CreateStaticIPRequest
		var awaited string
		diags := createVPCStaticIPWith(ctx, d, vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, pn string, req *client.CreateStaticIPRequest) (string, error) {
				if pn != "pn-1" {
					t.Fatalf("create must target the configured private network, got %q", pn)
				}
				gotReq = req
				return "act-create", nil
			},
			wait: func(ctx context.Context, activityID string) (*client.Activity, error) {
				awaited = activityID
				return completedActivity("si-new"), nil
			},
			read:       siRead(&client.StaticIP{ID: "si-new", Source: "custom", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			listStrict: noList,
		})
		if diags.HasError() {
			t.Fatalf("a clean create must succeed, got: %v", diags)
		}
		if awaited != "act-create" {
			t.Fatalf("the create's activity id must be threaded to wait, awaited %q", awaited)
		}
		if gotReq == nil {
			t.Fatal("create was never called")
		}
		if gotReq.MacAddress != "00:50:56:ab:cd:ef" || gotReq.IPAddress != "10.0.1.50" || gotReq.ResourceDescription != "seeded-desc" {
			t.Fatalf("create request must carry the configured fields, got %+v", gotReq)
		}
	})

	t.Run("a wait error fails closed with an actionable import diagnostic and no id", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		diags := createVPCStaticIPWith(ctx, d, vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, pn string, req *client.CreateStaticIPRequest) (string, error) {
				return "act-create", nil
			},
			wait: func(ctx context.Context, activityID string) (*client.Activity, error) {
				return nil, errors.New("activity failed")
			},
			read:       noRead,
			listStrict: noList,
		})
		if !diags.HasError() {
			t.Fatal("an unconfirmed creation (wait error) must fail closed, never report success")
		}
		foundImport := false
		for _, dg := range diags {
			if strings.Contains(dg.Summary, "import") {
				foundImport = true
			}
		}
		if !foundImport {
			t.Fatalf("the wait-error diagnostic must tell the operator to import the possibly-created static IP, got: %v", diags)
		}
		if d.Id() != "" {
			t.Fatalf("a wait error must leave the id empty (creation unconfirmed), got %q", d.Id())
		}
	})

	t.Run("a completed activity that reports no id fails closed before any read", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		diags := createVPCStaticIPWith(ctx, d, vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, pn string, req *client.CreateStaticIPRequest) (string, error) {
				return "act-create", nil
			},
			// State present (len 1) but empty Result -> setIdFromActivityState leaves id "".
			wait: func(ctx context.Context, activityID string) (*client.Activity, error) {
				return completedActivity(""), nil
			},
			read: func(ctx context.Context, id string) (*client.StaticIP, error) {
				t.Fatal("read must not be reached when the activity reported no id")
				return nil, nil
			},
			listStrict: noList,
		})
		if !diags.HasError() {
			t.Fatal("a completed activity with no id is a contract mismatch and must fail closed")
		}
		if d.Id() != "" {
			t.Fatalf("an empty activity result must not set an id, got %q", d.Id())
		}
	})

	t.Run("a completed activity sets the id and the create-mode read repopulates the state", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		desc := "fresh-from-read"
		diags := createVPCStaticIPWith(ctx, d, vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, pn string, req *client.CreateStaticIPRequest) (string, error) {
				return "act-create", nil
			},
			wait: func(ctx context.Context, activityID string) (*client.Activity, error) {
				return completedActivity("si-new"), nil
			},
			read: siRead(&client.StaticIP{
				ID:                  "si-new",
				IPAddress:           "10.0.1.77",
				MacAddress:          "00:50:56:ab:cd:ef",
				Source:              "custom",
				ResourceDescription: &desc,
				VPC:                 client.BaseObject{ID: "vpc-1"},
				PrivateNetwork:      client.BaseObject{ID: "pn-1"},
			}, nil),
			listStrict: noList,
		})
		if diags.HasError() {
			t.Fatalf("a completed create with a successful read must succeed, got: %v", diags)
		}
		if d.Id() != "si-new" {
			t.Fatalf("the id must come from the completed activity result, got %q", d.Id())
		}
		if got := d.Get("ip_address").(string); got != "10.0.1.77" {
			t.Fatalf("the create-mode read must repopulate the state from the live read, ip_address=%q", got)
		}
		if got := d.Get("resource_description").(string); got != "fresh-from-read" {
			t.Fatalf("the create-mode read must repopulate resource_description, got %q", got)
		}
	})

	t.Run("create mode never drops the fresh id on an eventually-consistent absence (B3)", func(t *testing.T) {
		d := newStaticIPCreateData(t)
		// The activity completed (positive creation evidence) and set the id, but the
		// immediate read is inconclusive (nil) and a complete 200 listing does NOT yet
		// contain si-new. This is eventual consistency, NOT a deletion: dropping the id
		// here would orphan the just-created static IP. The create path passes
		// dropOnConfirmedAbsence=false, so it MUST fail closed and PRESERVE the id.
		diags := createVPCStaticIPWith(ctx, d, vpcStaticIPCreateFuncs{
			create: func(ctx context.Context, pn string, req *client.CreateStaticIPRequest) (string, error) {
				return "act-create", nil
			},
			wait: func(ctx context.Context, activityID string) (*client.Activity, error) {
				return completedActivity("si-new"), nil
			},
			read:       siRead(nil, nil),
			listStrict: siListStrict(&client.StaticIP{ID: "other"}, &client.StaticIP{ID: "yet-another"}),
		})
		if !diags.HasError() {
			t.Fatal("a not-yet-visible just-created static IP must fail closed (eventual consistency), never be read as a deletion")
		}
		if d.Id() != "si-new" {
			t.Fatalf("create mode must PRESERVE the fresh id (never orphan it), got %q", d.Id())
		}
	})
}
