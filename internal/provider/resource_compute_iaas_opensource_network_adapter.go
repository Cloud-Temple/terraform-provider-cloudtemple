package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceOpenIaasNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		CreateContext: openIaasNetworkAdapterCreate,
		ReadContext:   openIaasNetworkAdapterRead,
		UpdateContext: openIaasNetworkAdapterUpdate,
		DeleteContext: openIaasNetworkAdapterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Description:  "The ID of the virtual machine to which the network adapter is attached.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"network_id": {
				Type:         schema.TypeString,
				Description:  "The ID of the network to which the network adapter is attached.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"mac_address": {
				Type:        schema.TypeString,
				Description: "The MAC address of the network adapter. If not specified, a random MAC address will be generated.",
				Optional:    true,
				Computed:    true,
			},
			"attached": {
				Type:        schema.TypeBool,
				Description: "Whether the network adapter is attached.",
				Optional:    true,
				Default:     true,
			},
			"tx_checksumming": {
				Type:        schema.TypeBool,
				Description: "Whether TX checksumming is enabled on the network adapter.",
				Optional:    true,
				Computed:    true,
			},

			// Out
			"id": {
				Type:        schema.TypeString,
				Description: "The ID of the network adapter.",
				Computed:    true,
			},
			"internal_id": {
				Type:        schema.TypeString,
				Description: "The internal ID of the network adapter.",
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the network adapter.",
				Computed:    true,
			},
			"machine_manager_id": {
				Type:        schema.TypeString,
				Description: "The ID of the machine manager of the network adapter.",
				Computed:    true,
			},
			"mtu": {
				Type:        schema.TypeInt,
				Description: "The MTU of the network adapter.",
				Computed:    true,
			},
			"ipv4_address": {
				Type:        schema.TypeString,
				Description: "The IPv4 address of the network adapter.",
				Computed:    true,
			},
			"ipv6_address": {
				Type:        schema.TypeString,
				Description: "The IPv6 address of the network adapter.",
				Computed:    true,
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Description: "The ID of the VPC the network adapter is associated with, or an empty string when the adapter is not on a VPC network.",
				Computed:    true,
			},
			"vpc_name": {
				Type:        schema.TypeString,
				Description: "The name of the VPC the network adapter is associated with, or an empty string when the adapter is not on a VPC network.",
				Computed:    true,
			},
			"private_network_id": {
				Type:        schema.TypeString,
				Description: "The ID of the VPC private network the network adapter is associated with, or an empty string when the adapter is not on a VPC network.",
				Computed:    true,
			},
			"private_network_name": {
				Type:        schema.TypeString,
				Description: "The name of the VPC private network the network adapter is associated with, or an empty string when the adapter is not on a VPC network.",
				Computed:    true,
			},
			"static_ip_address": {
				Type:        schema.TypeString,
				Description: "The static IP address assigned to the network adapter on its VPC, or an empty string when the adapter is not on a VPC network.",
				Computed:    true,
			},
		},
	}
}

func openIaasNetworkAdapterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)
	vmID := d.Get("virtual_machine_id").(string)

	var activity *client.Activity
	var err error
	for attempt := 1; attempt <= maxTransientVIFAttempts; attempt++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return diag.Errorf("the network adapter could not be created (cancelled: %s)", ctxErr)
		}
		var activityId string
		activityId, err = c.Compute().OpenIaaS().NetworkAdapter().Create(ctx, &client.CreateOpenIaasNetworkAdapterRequest{
			VirtualMachineID: vmID,
			NetworkID:        d.Get("network_id").(string),
			MAC:              d.Get("mac_address").(string),
		})
		if err != nil {
			return diag.Errorf("the network adapter could not be created: %s", err)
		}
		activity, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err == nil {
			break
		}
		if !client.IsTransientActivityFailure(err) {
			// A permanent failure never seeds the state: the failed
			// activity's adapter id must not become the resource id.
			return diag.Errorf("the network adapter could not be created: %s", err)
		}
		// Anti-duplicate guard (#251, hardened by FF-1): the retry is only
		// allowed after a fully VERIFIED cleanup — the failed activity must
		// report exactly one adapter, confirmed present on this VM by the
		// strict listing, and its deletion must complete. Anything short of
		// that (silent activity, unconfirmed reference, listing
		// unavailable, delete refused or failed) aborts with an explicit
		// error: an unverified half-created adapter could be duplicated by
		// a new attempt.
		if cleanupErr := cleanupHalfCreatedVIFs(ctx, c, activity, vmID); cleanupErr != nil {
			return diag.Errorf("the network adapter creation failed transiently and the cleanup could not be confirmed, refusing to retry (check virtual machine %s manually): create failure: %s; cleanup failure: %s", vmID, err, cleanupErr)
		}
		if attempt == maxTransientVIFAttempts {
			break
		}
		tflog.Warn(ctx, fmt.Sprintf("create network adapter on %s: transient platform failure (attempt %d/%d), retrying: %s",
			vmID, attempt, maxTransientVIFAttempts, err))
		select {
		case <-ctx.Done():
			return diag.Errorf("the network adapter could not be created (cancelled while retrying: %s): %s", ctx.Err(), err)
		case <-time.After(time.Duration(attempt) * 10 * time.Second):
		}
	}
	if err != nil {
		// Final failure after a confirmed cleanup (or with no reported
		// adapter): never seed the state from a failed activity — its id
		// would point at an adapter that was just deleted (FF-1).
		return diag.Errorf("the network adapter could not be created: %s", err)
	}
	setIdFromActivityState(d, activity)
	if d.Id() == "" {
		// setIdFromActivityState is silent on a malformed activity state:
		// every post-create action below requires a real id (FF-1,
		// invariant 5 — a Disconnect on an empty id must never be sent).
		return diag.Errorf("the network adapter creation succeeded but the activity did not report the adapter id: refresh the state and import the adapter manually if needed")
	}

	if !d.Get("attached").(bool) {
		activityId, err := c.Compute().OpenIaaS().NetworkAdapter().Disconnect(ctx, d.Id())
		if err != nil {
			return diag.Errorf("the network adapter could not be detached: %s", err)
		}
		if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
			return diag.Errorf("failed to detech network adapter, %s", err)
		}
	}

	return openIaasNetworkAdapterUpdate(ctx, d, meta)
}

func openIaasNetworkAdapterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	// Récupérer l'adaptateur réseau par son ID
	networkAdapter, err := c.Compute().OpenIaaS().NetworkAdapter().Read(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if networkAdapter == nil {
		// The API answers 403 for unknown AND forbidden ids alike, and the
		// client maps both to nil: a deletion is only accepted under
		// strict listing evidence (#275 doctrine, FF-5).
		vmID := d.Get("virtual_machine_id").(string)
		scoped, err := c.Compute().OpenIaaS().NetworkAdapter().ListStrict(ctx, &client.OpenIaaSNetworkAdapterFilter{
			VirtualMachineID: vmID,
		})
		if err != nil {
			return diag.Errorf("network adapter %s could not be read and its deletion could not be confirmed: %s", d.Id(), err)
		}
		tenant, err := c.Compute().OpenIaaS().NetworkAdapter().ListStrict(ctx, &client.OpenIaaSNetworkAdapterFilter{})
		if err != nil {
			return diag.Errorf("network adapter %s could not be read and its deletion could not be confirmed: %s", d.Id(), err)
		}
		scopedIDs := map[string]bool{}
		for _, adapter := range scoped {
			if adapter != nil {
				scopedIDs[adapter.ID] = true
			}
		}
		tenantIDs := map[string]bool{}
		for _, adapter := range tenant {
			if adapter != nil {
				tenantIDs[adapter.ID] = true
			}
		}
		switch classifyMissingDevice(d.Id(), scopedIDs, tenantIDs) {
		case deviceStillInScope:
			return diag.Errorf("network adapter %s could not be read but is still listed on virtual machine %s: refusing to drop it from the state (possible access restriction)", d.Id(), vmID)
		case deviceExistsOutOfScope:
			return diag.Errorf("network adapter %s could not be read and is no longer attached to virtual machine %s but still exists platform-side: refusing to treat this drift as a deletion — refresh or import after fixing the attachment", d.Id(), vmID)
		}
		// Deletion confirmed by independent strict reads.
		d.SetId("")
		return nil
	}

	// Mapper les données en utilisant la fonction helper
	adapterData := helpers.FlattenOpenIaaSNetworkAdapter(networkAdapter)

	// Définir les données dans le state
	for k, v := range adapterData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func openIaasNetworkAdapterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)

	if d.HasChange("network_id") || d.HasChange("mac_address") || d.HasChange("tx_checksumming") {
		// At create time every HasChange is true while the Create request
		// already carried network_id and mac: compare against the live
		// adapter and only push real divergences. The redundant PATCH was
		// rejected platform-side as a Static IP self-conflict ("MAC address
		// is already used by virtual machine <the adapter's own VM>"),
		// failing otherwise healthy multi-NIC provisioning (#246).
		adapter, err := c.Compute().OpenIaaS().NetworkAdapter().Read(ctx, d.Id())
		if err != nil {
			return diag.Errorf("failed to read network adapter: %s", err)
		}
		if adapter == nil {
			return diag.Errorf("network adapter %s not found", d.Id())
		}
		networkID := d.Get("network_id").(string)
		mac := d.Get("mac_address").(string)
		txChecksumming := d.Get("tx_checksumming").(bool)
		// tx_checksumming is Optional+Computed and absent from the Create
		// request: push it whenever it is EXPLICITLY configured and diverges
		// from the live value (also covers the first apply).
		txConfigured := false
		if raw := d.GetRawConfig(); !raw.IsNull() {
			if v := raw.GetAttr("tx_checksumming"); !v.IsNull() {
				txConfigured = true
			}
		}
		// Payload limited to the fields that actually diverge from the live
		// adapter: re-sending the current networkId/mac is rejected
		// platform-side as a VPC Static IP self-conflict (#246). The builder
		// is re-evaluated against a fresh read before every retry attempt.
		buildPatch := func(actual *client.OpenIaaSNetworkAdapter) *client.UpdateOpenIaasNetworkAdapterRequest {
			req := &client.UpdateOpenIaasNetworkAdapterRequest{}
			if networkID != "" && networkID != actual.Network.ID {
				req.NetworkID = networkID
			}
			if mac != "" && !strings.EqualFold(mac, actual.MacAddress) {
				req.MAC = mac
			}
			if txConfigured && txChecksumming != actual.TxChecksumming {
				req.TxChecksumming = &txChecksumming
			}
			if req.NetworkID == "" && req.MAC == "" && req.TxChecksumming == nil {
				return nil
			}
			return req
		}
		if buildPatch(adapter) != nil {
			// Bounded retry on transient platform failures (#251).
			if err := runVIFUpdateWithRetry(ctx, d.Id(), clientVIFUpdateFuncs(c, d.Id(), getWaiterOptions(ctx)), buildPatch); err != nil {
				return diag.Errorf("the network adapter could not be updated: %s", err)
			}
		}
	}

	if d.HasChange("attached") && !d.IsNewResource() {
		switch d.Get("attached").(bool) {
		case true:
			activityId, err := c.Compute().OpenIaaS().NetworkAdapter().Connect(ctx, d.Id())
			if err != nil {
				return diag.Errorf("the network adapter could not be attached: %s", err)
			}
			if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
				return diag.Errorf("failed to attach network adapter, %s", err)
			}
		case false:
			activityId, err := c.Compute().OpenIaaS().NetworkAdapter().Disconnect(ctx, d.Id())
			if err != nil {
				return diag.Errorf("the network adapter could not be detached: %s", err)
			}
			if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
				return diag.Errorf("failed to detach network adapter, %s", err)
			}
		}
	}

	return openIaasNetworkAdapterRead(ctx, d, meta)
}

func openIaasNetworkAdapterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().OpenIaaS().NetworkAdapter().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete netork adapter: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete network adapter, %s", err)
	}
	return nil
}

// vifCleanupTargets partitions the adapter ids referenced by the failed
// create activity: ids confirmed present on this VM by the strict listing
// must be deleted; referenced ids ABSENT from the listing are UNCONFIRMED.
// By attribution the ConcernedItems of OUR create activity are ours, so an
// absence from the listing — which may lag the platform state right after
// a transient incident — is an inconsistency, never a green light: an
// unconfirmed id forbids the retry (fail closed, #275 doctrine).
func vifCleanupTargets(failed *client.Activity, listedAdapterIDs map[string]bool) (toDelete []string, unconfirmed []string) {
	if failed == nil {
		return nil, nil
	}
	for _, item := range failed.ConcernedItems {
		if item.Type != "network_adapter" || item.ID == "" {
			continue
		}
		if listedAdapterIDs[item.ID] {
			toDelete = append(toDelete, item.ID)
		} else {
			unconfirmed = append(unconfirmed, item.ID)
		}
	}
	return toDelete, unconfirmed
}

// cleanupHalfCreatedVIFs deletes the adapter referenced by THIS create's
// failed activity, under strict evidence, and returns an error whenever the
// cleanup cannot be CONFIRMED: retrying after an unconfirmed cleanup could
// duplicate the adapter on the VM (#251, FF-1). The retry is only allowed
// when the platform reported what the failed create touched AND the
// cleanup is fully verified — a silent activity forbids the retry: an
// unreported half-created adapter could otherwise be duplicated.
func cleanupHalfCreatedVIFs(ctx context.Context, c *client.Client, failed *client.Activity, vmID string) error {
	if failed == nil {
		return fmt.Errorf("the platform did not report the failed create activity: the cleanup cannot be verified")
	}
	adapters, err := c.Compute().OpenIaaS().NetworkAdapter().ListStrict(ctx, &client.OpenIaaSNetworkAdapterFilter{
		VirtualMachineID: vmID,
	})
	if err != nil {
		return fmt.Errorf("could not list the adapters of virtual machine %s to confirm the cleanup: %w", vmID, err)
	}
	listed := map[string]bool{}
	for _, adapter := range adapters {
		listed[adapter.ID] = true
	}
	toDelete, unconfirmed := vifCleanupTargets(failed, listed)
	if len(unconfirmed) > 0 {
		return fmt.Errorf("the failed activity references adapter(s) %s that the strict listing of virtual machine %s does not confirm: the cleanup cannot be verified", strings.Join(unconfirmed, ", "), vmID)
	}
	if len(toDelete) == 0 {
		// No adapter reported at all: nothing can be safely cleaned up,
		// and an unreported half-created adapter could be duplicated by a
		// new attempt.
		return fmt.Errorf("the failed activity reports no adapter on virtual machine %s: the cleanup cannot be verified", vmID)
	}
	if len(toDelete) > 1 {
		// A create produces exactly one adapter: an activity referencing
		// several confirmed adapters cannot be attributed safely.
		return fmt.Errorf("the failed activity references %d adapters (%s) on virtual machine %s: the cleanup cannot be attributed to this create", len(toDelete), strings.Join(toDelete, ", "), vmID)
	}
	for _, id := range toDelete {
		delActivity, err := c.Compute().OpenIaaS().NetworkAdapter().Delete(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete the half-created adapter %s: %w", id, err)
		}
		if _, err := c.Activity().WaitForCompletion(ctx, delActivity, getWaiterOptions(ctx)); err != nil {
			return fmt.Errorf("failed to confirm the deletion of the half-created adapter %s: %w", id, err)
		}
	}
	return nil
}
