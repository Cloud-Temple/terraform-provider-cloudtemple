package provider

import (
	"context"
	"errors"
	"net/http"
	"regexp"

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

	// Create is SYNCHRONOUS: the API returns 201 with the new id in the body
	// (static_ip_id), NOT an activity. The client decodes that id.
	id, err := c.VPC().StaticIP().Create(ctx, d.Get("private_network_id").(string), &client.CreateStaticIPRequest{
		MacAddress:          d.Get("mac_address").(string),
		IPAddress:           d.Get("ip_address").(string),
		ResourceDescription: d.Get("resource_description").(string),
	})
	if err != nil {
		return diag.Errorf("failed to create VPC static IP: %s", err)
	}

	// Set the id immediately, before the read, so a later failure cannot orphan
	// the just-created static IP outside the Terraform state.
	d.SetId(id)

	return resourceVPCStaticIPRead(ctx, d, meta)
}

func resourceVPCStaticIPRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return readVPCStaticIPInto(
		ctx, d,
		c.VPC().StaticIP().Read,
		func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
			return c.VPC().StaticIP().ListStrict(ctx, privateNetworkID)
		},
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
//   - id confirmed ABSENT from a complete 200 listing -> this is genuine deletion
//     evidence (the listing is scoped to the static IP's own private network and
//     is provably complete), so the resource is dropped with SetId("").
//
// The private_network_id needed to scope the listing comes from the state; if it
// is empty the listing cannot be scoped, so we FAIL CLOSED rather than drop.
func readVPCStaticIPInto(ctx context.Context, d *schema.ResourceData, read vpcStaticIPReadFunc, listStrict vpcStaticIPListStrictFunc) diag.Diagnostics {
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
		// Confirmed absent from a complete 200 listing of its own private network:
		// genuine deletion evidence. Drop the resource.
		d.SetId("")
		return nil
	}

	sw := newStateWriter(d)
	for k, v := range helpers.FlattenStaticIP(staticIP) {
		sw.set(k, v)
	}
	return sw.diags
}

func resourceVPCStaticIPUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Diff-driven PATCH body: send ONLY the changed updatable fields. The
	// UpdateStaticIpPayload schema contains exactly resourceDescription and
	// macAddress (no ipAddress), so only those can change here; ip_address and
	// private_network_id are ForceNew and never reach this path.
	req := &client.UpdateStaticIPRequest{}
	changed := false
	if d.HasChange("mac_address") {
		v := d.Get("mac_address").(string)
		req.MacAddress = &v
		changed = true
	}
	if d.HasChange("resource_description") {
		v := d.Get("resource_description").(string)
		req.ResourceDescription = &v
		changed = true
	}

	if changed {
		// Update is ASYNCHRONOUS: the PATCH returns an activity (Location); wait
		// for its completion before reading back.
		activityID, err := c.VPC().StaticIP().Update(ctx, d.Id(), req)
		if err != nil {
			return diag.Errorf("failed to update VPC static IP %s: %s", d.Id(), err)
		}
		if _, err := c.Activity().WaitForCompletion(ctx, activityID, getWaiterOptions(ctx)); err != nil {
			return diag.Errorf("failed to update VPC static IP %s: %s", d.Id(), err)
		}
	}

	return resourceVPCStaticIPRead(ctx, d, meta)
}

func resourceVPCStaticIPDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Delete is ASYNCHRONOUS: the DELETE returns an activity (Location). A 404
	// (already gone) is success (idempotent).
	activityID, err := c.VPC().StaticIP().Delete(ctx, d.Id())
	if err != nil {
		if isNotFoundStatus(err) {
			return nil
		}
		return diag.Errorf("failed to delete VPC static IP %s: %s", d.Id(), err)
	}
	if _, err := c.Activity().WaitForCompletion(ctx, activityID, getWaiterOptions(ctx)); err != nil {
		if isNotFoundStatus(err) {
			return nil
		}
		return diag.Errorf("failed to delete VPC static IP %s: %s", d.Id(), err)
	}

	// Confirm absence via a strict, complete (200-only) listing of the private
	// network before returning. If the listing fails (non-200, transport), we do
	// not block the delete on an unprovable signal, but a still-present id is a
	// real failure that must surface.
	privateNetworkID := d.Get("private_network_id").(string)
	if privateNetworkID == "" {
		return nil
	}
	list, lerr := c.VPC().StaticIP().ListStrict(ctx, privateNetworkID)
	if lerr != nil {
		// The delete activity already completed; an unprovable listing must not
		// resurrect the resource. The completed activity is the primary signal.
		return nil
	}
	for _, si := range list {
		if si != nil && si.ID == d.Id() {
			return diag.Errorf(
				"VPC static IP %s is still listed on private network %s after its delete activity completed; the deletion could not be confirmed.",
				d.Id(), privateNetworkID,
			)
		}
	}

	return nil
}

// isNotFoundStatus reports whether err is an HTTP 404 from the API, used to make
// the delete idempotent.
func isNotFoundStatus(err error) bool {
	var statusErr client.StatusError
	return errors.As(err, &statusErr) && statusErr.Code == http.StatusNotFound
}
