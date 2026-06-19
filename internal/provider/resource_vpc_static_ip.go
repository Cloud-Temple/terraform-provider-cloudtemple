package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// macAddressRegexp matches the API MAC format xx:xx:xx:xx:xx:xx (also tolerating
// dashes), mirroring the swagger pattern on CreateStaticIp/UpdateStaticIpPayload.
var macAddressRegexp = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)

func resourceVPCStaticIP() *schema.Resource {
	return &schema.Resource{
		Description: "Allocate and manage a VPC static IP on a private network, bound to a virtual machine network adapter MAC address.",

		CreateContext: resourceVPCStaticIPCreate,
		ReadContext:   resourceVPCStaticIPRead,
		UpdateContext: resourceVPCStaticIPUpdate,
		DeleteContext: resourceVPCStaticIPDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"private_network_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the private network on which to allocate the static IP. Changing this forces a new resource.",
			},
			"mac_address": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(macAddressRegexp, "must be a MAC address in the format xx:xx:xx:xx:xx:xx"),
				Description:  "The MAC address of the network adapter bound to this static IP. Mutable: updating it issues a PATCH on the static IP.",
			},
			"ip_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The static IP address. Optional: if omitted it is auto-assigned by the API. The address is not updatable; changing it forces a new resource.",
			},
			"resource_description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An optional description of the static IP resource. Mutable via a PATCH on the static IP.",
			},

			// Out — mirrors the cloudtemple_vpc_static_ip datasource read-back
			// (IDs only, no *_name fields), populated by helpers.FlattenStaticIP.
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the static IP.",
			},
			"source": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The source of the static IP (one of `xoa`, `vmware`, `custom`).",
			},
			"virtual_machine_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the virtual machine associated with this static IP, if any.",
			},
			"network_adapter_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the network adapter associated with this static IP, if any.",
			},
			"floating_ip_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the floating IP bound to this static IP, if any.",
			},
			"floating_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The address of the floating IP bound to this static IP, if any.",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the VPC this static IP belongs to.",
			},
		},
	}
}

func resourceVPCStaticIPCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return createVPCStaticIPWith(ctx, d, vpcStaticIPCreateFuncs{
		create: c.VPC().StaticIP().Create,
		wait: func(ctx context.Context, activityID string) (*client.Activity, error) {
			return c.Activity().WaitForCompletion(ctx, activityID, getWaiterOptions(ctx))
		},
		read: c.VPC().StaticIP().Read,
		listStrict: func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
			return c.VPC().StaticIP().ListStrict(ctx, privateNetworkID)
		},
	})
}

// vpcStaticIPCreateFuncs abstracts the static IP create API surface so the create
// orchestration is unit tested without HTTP calls. wait returns the COMPLETED
// activity so the new id can be read from its state result.
type vpcStaticIPCreateFuncs struct {
	create     func(ctx context.Context, privateNetworkID string, req *client.CreateStaticIPRequest) (string, error)
	wait       func(ctx context.Context, activityID string) (*client.Activity, error)
	read       vpcStaticIPReadFunc
	listStrict vpcStaticIPListStrictFunc
}

// createVPCStaticIPWith holds the testable create logic. Create is ASYNCHRONOUS:
// the POST returns an activity (Location), and the COMPLETED activity carries the
// new static IP id in its state result (read via setIdFromActivityState). State
// safety drives every failure mode — the worst outcome is an ORPHAN (created
// platform-side, absent from the state):
//
//   - create error -> fail; nothing was confirmed created (no id is set).
//   - wait error -> FAIL CLOSED with an actionable diagnostic: the static IP MAY
//     have been created, so the message tells the operator to import or release it
//     before re-applying, to avoid a duplicate.
//   - activity completed but reported no id -> FAIL CLOSED: the id could not be
//     learned so the resource cannot be tracked; surface a contract-mismatch
//     diagnostic rather than SetId("") (which would silently orphan it).
//   - id set -> read back in CREATE mode (dropOnConfirmedAbsence=false): a
//     just-created static IP not yet visible in the listing is eventual
//     consistency, never a deletion, so the read NEVER drops the fresh id (#348).
func createVPCStaticIPWith(ctx context.Context, d *schema.ResourceData, funcs vpcStaticIPCreateFuncs) diag.Diagnostics {
	privateNetworkID := d.Get("private_network_id").(string)
	mac := d.Get("mac_address").(string)

	activityID, err := funcs.create(ctx, privateNetworkID, &client.CreateStaticIPRequest{
		MacAddress:          mac,
		IPAddress:           d.Get("ip_address").(string),
		ResourceDescription: d.Get("resource_description").(string),
	})
	if err != nil {
		return diag.Errorf("failed to create VPC static IP on private network %s (MAC %s): %s", privateNetworkID, mac, err)
	}

	activity, err := funcs.wait(ctx, activityID)
	if err != nil {
		return diag.Errorf(
			"VPC static IP creation could not be confirmed (activity %s, private network %s, MAC %s): %s. If the static IP was created, import it (terraform import) or release it platform-side before re-applying, to avoid a duplicate.",
			activityID, privateNetworkID, mac, err,
		)
	}

	setIdFromActivityState(d, activity)
	if d.Id() == "" {
		return diag.Errorf(
			"VPC static IP creation activity %s completed but reported no static IP id (private network %s, MAC %s); the static IP may exist platform-side. This is a provider/API contract mismatch — import it manually if it was created.",
			activityID, privateNetworkID, mac,
		)
	}

	// Create mode: NEVER drop the just-created id on an eventually-consistent read.
	return readVPCStaticIPInto(ctx, d, funcs.read, funcs.listStrict, false)
}

func resourceVPCStaticIPRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return readVPCStaticIPInto(
		ctx, d,
		c.VPC().StaticIP().Read,
		func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
			return c.VPC().StaticIP().ListStrict(ctx, privateNetworkID)
		},
		true, // refresh: a confirmed absence is genuine deletion evidence -> drop.
	)
}

// vpcStaticIPReadFunc and vpcStaticIPListStrictFunc abstract the static IP API
// surface used by readVPCStaticIPInto so the read logic is unit tested without
// HTTP calls.
type vpcStaticIPReadFunc func(ctx context.Context, id string) (*client.StaticIP, error)
type vpcStaticIPListStrictFunc func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error)

// readVPCStaticIPInto holds the testable read logic. State safety is the
// overriding invariant: the resource is NEVER dropped on an inconclusive read.
//
// A nil per-id read is INCONCLUSIVE, not a deletion: the VPC read client maps
// HTTP 403 to nil (the #303 access-denied-as-not-found convention), so an
// access-denied answer is indistinguishable from a genuine absence. We therefore
// confirm via a strict, complete (200-only) listing of the private network
// before concluding anything:
//
//   - listing error (includes any non-200 rejected by ListStrict, e.g. a 206
//     partial, a 403, or a 5xx) -> FAIL CLOSED: keep the resource, error;
//   - id still present -> the per-id read was a transient/permission blip but the
//     static IP exists -> FAIL CLOSED: keep the resource, error (its mutable
//     attributes could not be refreshed, so we must not report a clean refresh);
//   - id confirmed ABSENT from a complete 200 listing -> genuine deletion evidence
//     (the listing is scoped to the static IP's own private network and is
//     provably complete). What happens then depends on dropOnConfirmedAbsence:
//     on the REFRESH path (true) the resource is dropped with SetId(""); on the
//     CREATE path (false) it FAILS CLOSED instead, because right after a completed
//     create activity an absent listing is eventual consistency, not a deletion —
//     dropping a just-created id would SetId("") and orphan the static IP (#348).
//
// The private_network_id needed to scope the listing comes from the state; if it
// is empty the listing cannot be scoped, so we FAIL CLOSED rather than drop.
func readVPCStaticIPInto(ctx context.Context, d *schema.ResourceData, read vpcStaticIPReadFunc, listStrict vpcStaticIPListStrictFunc, dropOnConfirmedAbsence bool) diag.Diagnostics {
	id := d.Id()

	staticIP, err := read(ctx, id)
	if err != nil {
		return diag.Errorf("failed to read VPC static IP %s: %s", id, err)
	}

	if staticIP == nil {
		privateNetworkID := d.Get("private_network_id").(string)
		if privateNetworkID == "" {
			return diag.Errorf(
				"VPC static IP %s could not be read and its existence could not be confirmed because the private_network_id is missing from the state; the resource is kept in the state to avoid a wrong deletion. Resolve the read error, then refresh or re-import it.",
				id,
			)
		}

		list, lerr := listStrict(ctx, privateNetworkID)
		if lerr != nil {
			// A non-200 (206 partial, 403, 5xx) or a transport error cannot prove
			// absence. Keep the resource and fail closed.
			return diag.Errorf(
				"VPC static IP %s could not be read and its existence could not be confirmed (the strict listing of private network %s failed); the resource is kept in the state to avoid a wrong deletion: %s",
				id, privateNetworkID, lerr,
			)
		}
		for _, si := range list {
			if si != nil && si.ID == id {
				// Still present: the per-id read failed transiently or by access
				// restriction. Refuse to drop, and refuse to report a clean
				// refresh (the mutable attributes could not be re-read).
				return diag.Errorf(
					"VPC static IP %s could not be read but is still listed on private network %s; the resource is kept in the state (refusing to drop it on a likely transient error or access restriction). Its attributes could not be refreshed — retry once the read succeeds.",
					id, privateNetworkID,
				)
			}
		}
		// Confirmed absent from a complete 200 listing of its own private network.
		if !dropOnConfirmedAbsence {
			// Create path: the static IP was JUST created (the create activity
			// completed — positive creation evidence) and its id was set from that
			// activity. A complete listing that does not YET contain it is eventual
			// consistency, NOT a deletion: dropping the id here would SetId("") and
			// orphan the fresh static IP (created platform-side, absent from the
			// state). Fail closed instead, keeping the id so Terraform still tracks
			// it; a re-apply/refresh repopulates its attributes (#348).
			return diag.Errorf(
				"VPC static IP %s was just created but is not yet visible in the strict listing of private network %s (eventual consistency); the resource is kept in the state with its id. Re-run `terraform apply` or `terraform refresh` to populate its attributes.",
				id, privateNetworkID,
			)
		}
		// Refresh path: a complete 200 listing of the static IP's own private
		// network that does not contain it is genuine deletion evidence. Drop it.
		d.SetId("")
		return nil
	}

	// Id-consistency guard: this was a by-id read (GET /static_ips/{id}), so the
	// returned object MUST carry exactly this id. An empty id, or a different id,
	// is a malformed/inconsistent response — trusting it would let FlattenStaticIP
	// write id="" (a drop) or rebind the state to the wrong static IP (an orphan).
	// Fail closed and keep the resource unchanged rather than refresh from it.
	if staticIP.ID != id {
		return diag.Errorf(
			"VPC static IP %s read returned a mismatched or empty id %q; refusing to refresh state on an inconsistent read (the resource is kept unchanged — retry once the read is consistent).",
			id, staticIP.ID,
		)
	}

	// Source guard (#311): this resource manages only CUSTOM static IPs — the ones
	// it allocates via POST. A platform-managed static IP (e.g. source "xoa",
	// auto-created when an adapter is attached to a VPC network) CANNOT be deleted
	// through the API, so Terraform must never adopt one: it would create a
	// resource it cannot destroy. Rejecting it here makes `terraform import` of a
	// non-custom static IP fail, and also surfaces a wrongly-adopted state on refresh.
	if staticIP.Source != "" && staticIP.Source != "custom" {
		return diag.Errorf(
			"VPC static IP %s has source %q; only \"custom\" static IPs (allocated by this resource) can be managed by Terraform. A platform-managed static IP cannot be deleted via the API — remove it from the state with `terraform state rm`.",
			id, staticIP.Source,
		)
	}

	sw := newStateWriter(d)
	for k, v := range helpers.FlattenStaticIP(staticIP) {
		sw.set(k, v)
	}
	return sw.diags
}

// vpcStaticIPUpdateFuncs abstracts the static IP update API surface so the
// PATCH logic is unit tested without HTTP calls or real sleeps. retrySleep /
// isTransient default to defaultVPCSleep / client.IsTransientActivityFailure.
type vpcStaticIPUpdateFuncs struct {
	read        vpcStaticIPReadFunc
	update      func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error)
	wait        vpcActivityWaitFunc
	retrySleep  func(ctx context.Context, attempt int) error
	isTransient func(err error) bool
}

func resourceVPCStaticIPUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	if diags := updateVPCStaticIPWith(ctx, d, vpcStaticIPUpdateFuncs{
		read:   c.VPC().StaticIP().Read,
		update: c.VPC().StaticIP().Update,
		wait: func(ctx context.Context, activityID string) error {
			_, err := c.Activity().WaitForCompletion(ctx, activityID, getWaiterOptions(ctx))
			return err
		},
	}); diags.HasError() {
		return diags
	}
	return resourceVPCStaticIPRead(ctx, d, meta)
}

// updateVPCStaticIPWith holds the testable update logic. The PATCH body
// (UpdateStaticIpPayload = exactly resourceDescription + macAddress; ip_address
// and private_network_id are ForceNew and never reach this path) is wrapped in
// the bounded transient-502 retry (#315/#319).
//
// Codex PLAN strict guard: the diff is rebuilt against a FRESH LIVE read before
// EVERY attempt, never from the state alone — so a retry after a transient
// failure that actually applied the change re-reads an already-converged static
// IP and returns success WITHOUT a second PATCH. A nil/ambiguous read (403/absent
// or an id-inconsistent body) FAILS CLOSED: never PATCH on ambiguous evidence.
func updateVPCStaticIPWith(ctx context.Context, d *schema.ResourceData, funcs vpcStaticIPUpdateFuncs) diag.Diagnostics {
	desiredMAC := d.Get("mac_address").(string)
	desiredDesc := d.Get("resource_description").(string)
	// Canonicalise the MAC for comparison (lowercase, ":"-separated), mirroring
	// the client's create-time matching, so a pure formatting difference between
	// the config and the live read never triggers a needless PATCH.
	normMAC := func(m string) string { return strings.ToLower(strings.ReplaceAll(m, "-", ":")) }

	updateErr := runVPCWriteWithRetry(ctx, vpcWriteRetry{
		label:       fmt.Sprintf("update VPC static IP %s", d.Id()),
		sleep:       funcs.retrySleep,
		isTransient: funcs.isTransient,
		attempt: func(ctx context.Context) error {
			live, rerr := funcs.read(ctx, d.Id())
			if rerr != nil {
				return rerr
			}
			if live == nil || live.ID != d.Id() {
				return fmt.Errorf("VPC static IP %s could not be read consistently before updating it; refusing to PATCH on ambiguous evidence", d.Id())
			}

			req := &client.UpdateStaticIPRequest{}
			changed := false
			if normMAC(desiredMAC) != normMAC(live.MacAddress) {
				v := desiredMAC
				req.MacAddress = &v
				changed = true
			}
			liveDesc := ""
			if live.ResourceDescription != nil {
				liveDesc = *live.ResourceDescription
			}
			if desiredDesc != liveDesc {
				v := desiredDesc
				req.ResourceDescription = &v
				changed = true
			}
			if !changed {
				// Live already matches the desired state: converged, no PATCH.
				return nil
			}

			// Update is ASYNCHRONOUS: the PATCH returns an activity (Location);
			// wait for its completion.
			activityID, uerr := funcs.update(ctx, d.Id(), req)
			if uerr != nil {
				return uerr
			}
			return funcs.wait(ctx, activityID)
		},
	})
	if updateErr != nil {
		return diag.Errorf("failed to update VPC static IP %s: %s", d.Id(), updateErr)
	}
	return nil
}

func resourceVPCStaticIPDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return deleteVPCStaticIPWith(
		ctx, d,
		c.VPC().StaticIP().Delete,
		func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
			return c.VPC().StaticIP().ListStrict(ctx, privateNetworkID)
		},
		func(ctx context.Context, activityID string) error {
			_, err := c.Activity().WaitForCompletion(ctx, activityID, getWaiterOptions(ctx))
			return err
		},
	)
}

// vpcStaticIPDeleteFunc and vpcActivityWaitFunc abstract the static IP delete API
// surface used by deleteVPCStaticIPWith so the delete logic is unit tested
// without HTTP calls.
type vpcStaticIPDeleteFunc func(ctx context.Context, id string) (activityID string, err error)
type vpcActivityWaitFunc func(ctx context.Context, activityID string) error

// deleteVPCStaticIPWith holds the testable delete logic. State safety is the
// overriding invariant: the resource is NEVER dropped from the state without
// positive evidence that the static IP is gone.
//
// Delete is ASYNCHRONOUS: a successful DELETE returns an activity (Location) and
// the COMPLETED activity is the deletion evidence (the established async-delete
// pattern: VMs, disks, buckets).
//
// The idempotency (already-absent) path is the delicate one. The VPC API reports
// absence ambiguously, so the two codes are NOT treated alike:
//
//   - 404 is UNAMBIGUOUS ("the resource you are deleting does not exist") -> the
//     static IP is already gone -> idempotent success.
//   - 403 is AMBIGUOUS: under the #303 convention the VPC API conflates "absent"
//     with "forbidden", so a 403 may be a genuine permission failure on a static
//     IP that still exists. Treating it as "absent" would SILENTLY DROP the
//     resource from the state on an authorization error — exactly the failure the
//     Read path refuses (it never drops on a 403/nil read without confirmation).
//     So a 403 is confirmed via a strict, complete (200-only) listing before the
//     delete is accepted; anything inconclusive FAILS CLOSED.
func deleteVPCStaticIPWith(ctx context.Context, d *schema.ResourceData, del vpcStaticIPDeleteFunc, listStrict vpcStaticIPListStrictFunc, wait vpcActivityWaitFunc) diag.Diagnostics {
	return deleteVPCStaticIPWithRetry(ctx, d, del, listStrict, wait, nil, nil)
}

// deleteVPCStaticIPWithRetry is deleteVPCStaticIPWith with the transient-502
// retry seams exposed for unit tests; retrySleep / isTransient default to
// defaultVPCSleep / client.IsTransientActivityFailure when nil.
//
// The DELETE+wait is wrapped in the bounded transient-502 retry (#315/#319); the
// 403-absence confirmation (confirmStaticIPDeleted) stays OUTSIDE it. No live
// re-read is needed between attempts: a re-DELETE is idempotent and the re-DELETE
// itself is the probe — a 404 proves the static IP is already gone (success).
func deleteVPCStaticIPWithRetry(ctx context.Context, d *schema.ResourceData, del vpcStaticIPDeleteFunc, listStrict vpcStaticIPListStrictFunc, wait vpcActivityWaitFunc, retrySleep func(context.Context, int) error, isTransient func(error) bool) diag.Diagnostics {
	// Preflight (#311): only CUSTOM static IPs are deletable via the API. If the
	// state already knows this is a platform-managed one (e.g. source "xoa"),
	// surface an actionable diagnostic instead of issuing a doomed delete. Import
	// is rejected in Read, so a TF-created resource is always custom; this is
	// defence in depth.
	if src, ok := d.Get("source").(string); ok && src != "" && src != "custom" {
		return diag.Errorf(
			"VPC static IP %s has source %q and cannot be deleted via the API (only \"custom\" static IPs are deletable). Remove it from the state with `terraform state rm`, or release it platform-side.",
			d.Id(), src,
		)
	}

	delErr := runVPCWriteWithRetry(ctx, vpcWriteRetry{
		label:       fmt.Sprintf("delete VPC static IP %s", d.Id()),
		sleep:       retrySleep,
		isTransient: isTransient,
		attempt: func(ctx context.Context) error {
			activityID, err := del(ctx, d.Id())
			if err != nil {
				// 404: unambiguous absence -> idempotent success.
				if isVPCStatusCode(err, http.StatusNotFound) {
					return nil
				}
				// 403 (ambiguous, #303) and any other error are non-transient:
				// surface them so the caller can apply the right rule below.
				return err
			}
			return wait(ctx, activityID)
		},
	})
	if delErr != nil {
		// 403: ambiguous (#303). Confirm absence before accepting the delete, so a
		// genuine permission error never silently drops the resource from the state.
		if isVPCStatusCode(delErr, http.StatusForbidden) {
			return confirmStaticIPDeleted(ctx, d, listStrict, delErr)
		}
		return diag.Errorf("failed to delete VPC static IP %s: %s", d.Id(), staticIPDeleteErrorDetail(delErr))
	}
	return nil
}

// confirmStaticIPDeleted is the 403-on-delete path. It accepts the delete as an
// idempotent success ONLY when a strict, complete (200-only) listing of the
// private network proves the static IP is absent. A still-present static IP, an
// inconclusive listing (any non-200 surfaced as an error by ListStrict), or a
// missing private_network_id scope all FAIL CLOSED with a diagnostic, so a
// forbidden delete can never silently remove the resource from the state.
func confirmStaticIPDeleted(ctx context.Context, d *schema.ResourceData, listStrict vpcStaticIPListStrictFunc, deleteErr error) diag.Diagnostics {
	privateNetworkID := d.Get("private_network_id").(string)
	if privateNetworkID == "" {
		return diag.Errorf(
			"VPC static IP %s delete returned 403 and its absence could not be confirmed because the private_network_id is missing from the state; the resource is kept to avoid a wrong removal. Resolve the access error, then retry. Original error: %s",
			d.Id(), deleteErr,
		)
	}
	list, lerr := listStrict(ctx, privateNetworkID)
	if lerr != nil {
		return diag.Errorf(
			"VPC static IP %s delete returned 403 and its absence could not be confirmed (the strict listing of private network %s failed); the resource is kept in the state: %s (original delete error: %s)",
			d.Id(), privateNetworkID, lerr, deleteErr,
		)
	}
	for _, si := range list {
		if si != nil && si.ID == d.Id() {
			return diag.Errorf(
				"VPC static IP %s could not be deleted (the API returned 403) and it is still present on private network %s; the resource is kept in the state. This is an authorization failure, not a deletion — check the token's permissions and retry.",
				d.Id(), privateNetworkID,
			)
		}
	}
	// Confirmed absent from a complete 200 listing of its own private network:
	// genuine idempotent success.
	return nil
}

// isVPCStatusCode reports whether err is a client.StatusError with the given code.
func isVPCStatusCode(err error, code int) bool {
	var statusErr client.StatusError
	return errors.As(err, &statusErr) && statusErr.Code == code
}

// staticIPDeleteErrorDetail turns the platform's "not a custom static IP"
// refusal into an actionable message; otherwise it returns the raw error text.
func staticIPDeleteErrorDetail(err error) string {
	if strings.Contains(err.Error(), "not a custom static IP") {
		return "the platform refuses the delete because it is not a \"custom\" static IP (it is platform-managed, e.g. source \"xoa\"); remove it from the state with `terraform state rm` or release it platform-side"
	}
	return err.Error()
}
