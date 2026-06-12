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
		},
	}
}

func openIaasNetworkAdapterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)
	vmID := d.Get("virtual_machine_id").(string)

	var activity *client.Activity
	var err error
	for attempt := 1; attempt <= maxTransientVIFAttempts; attempt++ {
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
		if err == nil || !client.IsTransientActivityFailure(err) {
			break
		}
		// Anti-duplicate guard (#251): only delete the VIF referenced by the
		// FAILED activity itself (ConcernedItems) — never by listing diff,
		// another actor may legitimately create VIFs on the same VM in the
		// meantime. A half-created VIF has an uncertain VPC registration:
		// cleaning it is safer than adopting it. If the failed activity
		// references no adapter, nothing is deleted and the retry may leave
		// an orphan VIF platform-side: accepted trade-off — the platform did
		// not report what it created, and a wrong deletion would be worse.
		if activity != nil {
			for _, item := range activity.ConcernedItems {
				if item.Type == "network_adapter" && item.ID != "" {
					if delActivity, derr := c.Compute().OpenIaaS().NetworkAdapter().Delete(ctx, item.ID); derr == nil {
						_, _ = c.Activity().WaitForCompletion(ctx, delActivity, getWaiterOptions(ctx))
					}
				}
			}
		}
		if attempt == maxTransientVIFAttempts {
			break
		}
		tflog.Warn(ctx, fmt.Sprintf("create network adapter on %s: transient platform failure (attempt %d/%d), retrying: %s",
			vmID, attempt, maxTransientVIFAttempts, err))
		select {
		case <-ctx.Done():
			return diag.Errorf("the network adapter could not be created: %s", err)
		case <-time.After(time.Duration(attempt) * 10 * time.Second):
		}
	}
	setIdFromActivityState(d, activity)
	if err != nil {
		return diag.Errorf("the network adapter could not be created: %s", err)
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
		d.SetId("") // L'adaptateur n'existe plus, marquer la ressource comme supprimée
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
