package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// NOTE (v1.9.0 rebuild, C4): a floating IP is a BILLABLE public IP. Provision is
// ASYNCHRONOUS (POST /vpc/v1/floating_ips {"count":1} -> 201 + Location; the id is
// read from the completed activity state — proven live, ~85s). Deprovision is async
// too and GATED BY CONSTRUCTION in the client (DeprovisionUnbound: prove unbound ->
// delete -> POSITIVE 404 confirm), so destroying a BOUND floating IP is refused
// rather than silently breaking a binding. The overriding invariant on every path
// is §5: never silently ORPHAN a billable IP (created platform-side, absent from
// state) and never falsely report success.
//
// SCOPE (C4): this resource manages the UNBOUND lifecycle — provision, describe,
// deprovision. Binding to a static IP is intentionally NOT managed here (the
// static_ip_*/vpc_id/private_network_id attributes are read-only): the client's
// only deletion path refuses a bound IP, so binding belongs to a dedicated
// resource/chunk. There is deliberately NO transient-status retry on the write
// paths (same rationale as the static IP resource): the doctrine is FAIL CLOSED,
// not retry — a failed activity surfaces an actionable diagnostic and the operator
// re-applies (Terraform's natural retry).

func resourceVPCFloatingIP() *schema.Resource {
	return &schema.Resource{
		Description: "Provision and manage a VPC floating (public) IP. The IP is allocated unbound; binding it to a static IP is not managed by this resource (those attributes are read-only). Destroying the resource deprovisions the billable IP — which is refused while it is bound.",

		CreateContext: resourceVPCFloatingIPCreate,
		ReadContext:   resourceVPCFloatingIPRead,
		UpdateContext: resourceVPCFloatingIPUpdate,
		DeleteContext: resourceVPCFloatingIPDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		// Generous timeouts: provision/deprovision are async and a billable IP must
		// never be abandoned mid-flight by a premature timeout (the create sets the id
		// BEFORE any other call, so even a timeout cannot orphan it outside state).
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Update: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			// In — the ONLY settable field. Provision itself takes no input (the live
			// contract is POST {"count":1}); the description is applied via a follow-up
			// PATCH. Optional+Computed: when omitted, the API-assigned value is kept
			// (no perpetual diff); when set, a non-whitespace value is required.
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "A description for the floating IP. Optional and mutable: setting or changing it issues a PATCH on the floating IP. When omitted, the API-assigned value is kept.",
			},

			// Out — mirrors the cloudtemple_vpc_floating_ip datasource, populated by
			// helpers.FlattenFloatingIP. static_ip_*/vpc_id/private_network_id reflect
			// a binding made elsewhere and are READ-ONLY here.
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the floating IP.",
			},
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The public IP address allocated for this floating IP.",
			},
			"static_ip_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the static IP this floating IP is bound to, if any. Read-only: binding is not managed by this resource.",
			},
			"static_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The address of the static IP this floating IP is bound to, if any.",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the VPC this floating IP is associated with when bound.",
			},
			"private_network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the private network this floating IP is associated with when bound.",
			},
		},
	}
}

// floatingIPReadMode selects how readVPCFloatingIPInto treats an AUTHORITATIVE 404
// (the floating IP's SOLE absence channel — ResolveByID maps only a by-id 404 to
// not-found, never a 403/206, so absence is always provable):
//   - floatingIPReadForRefresh: a 404 is genuine deletion evidence -> drop (SetId(""));
//   - floatingIPReadAfterWrite: right after a provision/PATCH the IP's existence is
//     POSITIVE evidence, so a 404 is eventual consistency, NOT a deletion -> FAIL
//     CLOSED keeping the id (never orphan a just-provisioned BILLABLE IP).
type floatingIPReadMode int

const (
	floatingIPReadForRefresh floatingIPReadMode = iota
	floatingIPReadAfterWrite
)

// vpcFloatingIPResolveFunc abstracts the strict tri-state by-id read (ResolveByID)
// so the read/create/update logic is unit tested without HTTP calls.
type vpcFloatingIPResolveFunc func(ctx context.Context, id string) (*client.FloatingIP, bool, error)

// vpcFloatingIPCreateFuncs abstracts the create API surface so the create
// orchestration is unit tested without HTTP calls. provision returns the new id:
// the client (Provision = ProvisionStart + WaitProvision) owns the async id
// resolution and FAILS CLOSED (it never returns ("", nil); a wait failure is
// wrapped WITH the activityID), so the provider only has to honour that contract.
type vpcFloatingIPCreateFuncs struct {
	provision         func(ctx context.Context) (string, error)
	updateDescription func(ctx context.Context, fipID, description string) (activityID string, err error)
	wait              vpcActivityWaitFunc
	resolve           vpcFloatingIPResolveFunc
}

func resourceVPCFloatingIPCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return createVPCFloatingIPWith(ctx, d, vpcFloatingIPCreateFuncs{
		provision: func(ctx context.Context) (string, error) {
			return c.VPC().FloatingIP().Provision(ctx, getWaiterOptions(ctx))
		},
		updateDescription: c.VPC().FloatingIP().UpdateDescription,
		wait: func(ctx context.Context, activityID string) error {
			_, err := c.Activity().WaitForCompletion(ctx, activityID, getWaiterOptions(ctx))
			return err
		},
		resolve: c.VPC().FloatingIP().ResolveByID,
	})
}

// createVPCFloatingIPWith holds the testable create logic. State safety drives every
// failure mode — the worst outcome is a BILLABLE ORPHAN (provisioned platform-side,
// absent from the state):
//
//   - provision error -> FAIL, never SetId. The id was never confirmed; the client
//     error already carries the activityID and a billing-audit hint (R-Q2).
//   - provision returned an empty id with no error -> FAIL CLOSED. The client must
//     never do this; guarding in depth means a contract regression can never
//     SetId("") and silently orphan a billable IP.
//   - id set BEFORE the optional description PATCH and the read-back, so neither can
//     orphan the just-provisioned IP outside the state.
//   - description PATCH failure is a WARNING, not an error: the (billable) IP is
//     provisioned and tracked, so we must NOT taint/recreate it over a cosmetic
//     label — the next apply reconciles the description via Update.
func createVPCFloatingIPWith(ctx context.Context, d *schema.ResourceData, funcs vpcFloatingIPCreateFuncs) diag.Diagnostics {
	id, err := funcs.provision(ctx)
	if err != nil {
		return diag.Errorf(
			"failed to provision VPC floating IP: %s. If an IP was allocated it is BILLABLE — audit the floating IPs for an unbound, recently-created one and import it (terraform import) or deprovision it before re-applying, to avoid a billable orphan.",
			err,
		)
	}
	if id == "" {
		return diag.Errorf(
			"VPC floating IP provisioning reported no id; a BILLABLE IP may have been allocated. This is a provider/API contract mismatch — audit the floating IPs and import it manually if it was created.",
		)
	}

	// Set the id BEFORE any further call, so a later failure (description PATCH or
	// read-back) can NEVER orphan the just-provisioned billable IP outside the state.
	d.SetId(id)

	var diags diag.Diagnostics

	// The provision contract takes no description; a configured one is a follow-up
	// PATCH. A PATCH failure does NOT fail the create (that would taint -> deprovision
	// a perfectly good billable IP over a label, then re-provision): keep the tracked
	// IP and surface a WARNING; the next apply reconciles the description via Update.
	if desc, ok := d.GetOk("description"); ok {
		if werr := patchFloatingIPDescription(ctx, funcs.updateDescription, funcs.wait, id, desc.(string)); werr != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Floating IP provisioned but its description could not be set",
				Detail: fmt.Sprintf(
					"The floating IP %s was provisioned and is tracked in the state (it is NOT orphaned), but applying its description failed: %s. Re-run `terraform apply` to reconcile the description.",
					id, werr,
				),
			})
		}
	}

	// readAfterWrite: a just-provisioned IP not yet visible by id is eventual
	// consistency, NEVER a deletion -> the read must not drop the fresh (billable) id.
	return append(diags, readVPCFloatingIPInto(ctx, d, funcs.resolve, floatingIPReadAfterWrite)...)
}

// patchFloatingIPDescription issues the description PATCH and waits for its activity
// ONLY when the PATCH was async. UpdateDescription returns a non-empty activity id
// for the async case and ("", nil) for a sync 2xx (no Location); waiting on an empty
// id would poll a non-existent activity and turn a success into a spurious failure.
// Shared by create (best-effort) and update (authoritative).
func patchFloatingIPDescription(ctx context.Context, update func(ctx context.Context, fipID, description string) (string, error), wait vpcActivityWaitFunc, id, description string) error {
	activityID, err := update(ctx, id, description)
	if err != nil {
		return err
	}
	if activityID == "" {
		// Sync PATCH success (2xx without a Location): there is no activity to wait on
		// (the UpdateDescription contract — the caller waits ONLY on a non-empty id).
		return nil
	}
	return wait(ctx, activityID)
}

func resourceVPCFloatingIPRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return readVPCFloatingIPInto(
		ctx, d,
		c.VPC().FloatingIP().ResolveByID,
		floatingIPReadForRefresh, // a confirmed (404) absence is genuine deletion evidence -> drop.
	)
}

// readVPCFloatingIPInto holds the testable read logic. State safety is the
// overriding invariant: the resource is NEVER dropped on an inconclusive read.
//
// ResolveByID is the STRICT by-id read used ONLY by the resource: unlike Read
// (datasources), it returns an explicit tri-state (plus an id-consistency guard)
// so the resource can act on each status distinctly. Its tri-state is therefore
// unambiguous:
//
//   - error (403/206/transport/other) -> FAIL CLOSED: keep the resource, error. A
//     forbidden answer is NOT proof of absence (#303); a partial/unexpected status
//     is never absence.
//   - found=false -> AUTHORITATIVE 404 (the sole absence channel): drop in refresh
//     mode; FAIL CLOSED in readAfterWrite mode (eventual consistency, not deletion).
//   - found=true -> the client already applied the id-consistency guard (a 200 body
//     with an empty/mismatched id fails closed inside ResolveByID), so the object
//     carries exactly this id -> flatten into state.
func readVPCFloatingIPInto(ctx context.Context, d *schema.ResourceData, resolve vpcFloatingIPResolveFunc, mode floatingIPReadMode) diag.Diagnostics {
	id := d.Id()

	fip, found, err := resolve(ctx, id)
	if err != nil {
		return diag.Errorf(
			"failed to read VPC floating IP %s: %s. The resource is kept in the state (a forbidden or partial response is not proof of absence); resolve the error, then refresh.",
			id, err,
		)
	}
	if !found {
		if mode == floatingIPReadAfterWrite {
			// Just provisioned/patched (positive existence evidence) and the id was set
			// from that write. An authoritative 404 here would be eventual consistency,
			// NOT a deletion: dropping the id would SetId("") and orphan the billable IP.
			// Fail closed, keeping the id. (Provision already waited for the activity to
			// complete, so this is anomalous and rare; re-apply/refresh repopulates it.)
			return diag.Errorf(
				"VPC floating IP %s was just written but is not yet visible on a by-id read (eventual consistency); the resource is kept in the state with its id. Re-run `terraform apply` or `terraform refresh` to populate its attributes.",
				id,
			)
		}
		// Refresh path: an authoritative 404 is genuine deletion evidence. Drop it.
		d.SetId("")
		return nil
	}

	sw := newStateWriter(d)
	for k, v := range helpers.FlattenFloatingIP(fip) {
		sw.set(k, v)
	}
	return sw.diags
}

// vpcFloatingIPUpdateFuncs abstracts the update API surface so the PATCH logic is
// unit tested without HTTP calls.
type vpcFloatingIPUpdateFuncs struct {
	resolve           vpcFloatingIPResolveFunc
	updateDescription func(ctx context.Context, fipID, description string) (activityID string, err error)
	wait              vpcActivityWaitFunc
}

func resourceVPCFloatingIPUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return updateVPCFloatingIPWith(ctx, d, vpcFloatingIPUpdateFuncs{
		resolve:           c.VPC().FloatingIP().ResolveByID,
		updateDescription: c.VPC().FloatingIP().UpdateDescription,
		wait: func(ctx context.Context, activityID string) error {
			_, err := c.Activity().WaitForCompletion(ctx, activityID, getWaiterOptions(ctx))
			return err
		},
	})
}

// updateVPCFloatingIPWith holds the testable update logic. Only the description is
// mutable (every other attribute is Computed and cannot change from config). The
// desired value is reconciled against a FRESH LIVE read, never the state alone:
//
//   - an inconclusive read (error) or an authoritative absence (404) FAILS CLOSED:
//     never PATCH on ambiguous evidence or a missing floating IP;
//   - if the live description already matches the desired one it is converged:
//     no PATCH is issued;
//   - otherwise issue the async description PATCH and wait for its activity;
//   - then read back through the SAME injected resolve (readAfterWrite) so the
//     post-write state is refreshed and the whole update path is seam-testable.
func updateVPCFloatingIPWith(ctx context.Context, d *schema.ResourceData, funcs vpcFloatingIPUpdateFuncs) diag.Diagnostics {
	desiredDesc := d.Get("description").(string)

	live, found, rerr := funcs.resolve(ctx, d.Id())
	if rerr != nil {
		return diag.Errorf("failed to update VPC floating IP %s: %s; refusing to PATCH on an inconclusive read", d.Id(), rerr)
	}
	if !found {
		return diag.Errorf(
			"failed to update VPC floating IP %s: it could not be found before updating (authoritative 404); refusing to PATCH a floating IP that is not present",
			d.Id(),
		)
	}
	// live.ID == d.Id() is guaranteed by ResolveByID's id-consistency guard.

	if desiredDesc != live.Description {
		if werr := patchFloatingIPDescription(ctx, funcs.updateDescription, funcs.wait, d.Id(), desiredDesc); werr != nil {
			return diag.Errorf("failed to update VPC floating IP %s description: %s", d.Id(), werr)
		}
	}

	// Post-write read-back through the same injected resolve. readAfterWrite: a
	// just-patched IP not yet visible on a by-id read is eventual consistency, NEVER
	// a deletion — the read must not drop the (billable) id. On the converged branch
	// this simply refreshes the computed attributes from the live read.
	return readVPCFloatingIPInto(ctx, d, funcs.resolve, floatingIPReadAfterWrite)
}

// vpcFloatingIPDeprovisionFunc abstracts the deletion path so the delete logic is
// unit tested without HTTP calls.
type vpcFloatingIPDeprovisionFunc func(ctx context.Context, fipID string) error

func resourceVPCFloatingIPDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return deleteVPCFloatingIPWith(ctx, d, func(ctx context.Context, fipID string) error {
		return c.VPC().FloatingIP().DeprovisionUnbound(ctx, fipID, getWaiterOptions(ctx))
	})
}

// deleteVPCFloatingIPWith holds the testable delete logic. The heavy lifting is in
// the client's DeprovisionUnbound, which is GATED AND CONFIRMED BY CONSTRUCTION:
//
//   - it reads the IP FRESH (resolveUnboundProof), so it gates on live evidence, not
//     the possibly-stale state — strictly safer than a state preflight here;
//   - a 404 is idempotent success (already gone), proven by a positive read;
//   - a clearly-bound IP gets a typed refusal ("unbind … before deprovisioning") and
//     any not-proven-unbound state FAILS CLOSED with NO delete issued;
//   - success ALWAYS rests on a final positive 404 confirmation, never a bare 2xx.
//
// So this resource path only has to surface the client's (already actionable) error.
func deleteVPCFloatingIPWith(ctx context.Context, d *schema.ResourceData, deprovision vpcFloatingIPDeprovisionFunc) diag.Diagnostics {
	if err := deprovision(ctx, d.Id()); err != nil {
		return diag.Errorf("failed to deprovision VPC floating IP %s: %s", d.Id(), err)
	}
	return nil
}
