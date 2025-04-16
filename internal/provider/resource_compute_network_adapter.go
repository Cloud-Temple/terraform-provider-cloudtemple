package provider

import (
	"context"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sethvargo/go-retry"
)

func resourceNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		Description: "",

		CreateWithoutTimeout: computeNetworkAdapterCreate,
		ReadContext:          computeNetworkAdapterRead,
		UpdateContext:        computeNetworkAdapterUpdate,
		DeleteContext:        computeNetworkAdapterDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the virtual machine to which the network adapter will be attached.",
			},
			"network_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the network to which the adapter will be connected.",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The type of the network adapter. Supported types are defined by the Guest OS, usually E1000 or VMXNET3.",
			},
			"mac_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The MAC address of the network adapter. If not provided, a MAC address will be generated automatically.",
			},
			"auto_connect": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the network adapter should connect to the network automatically when the virtual machine is powered on.",
			},
			"connected": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the network adapter should be connected to the network. Defaults to true. ",
			},

			// Out
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the network adapter.",
			},
			"mac_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of MAC address assignment. Possible values are MANUAL and GENERATED.",
			},
		},
	}
}

func computeNetworkAdapterCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().NetworkAdapter().Create(ctx, &client.CreateNetworkAdapterRequest{
		VirtualMachineId: d.Get("virtual_machine_id").(string),
		MacAddress:       d.Get("mac_address").(string),
		NetworkId:        d.Get("network_id").(string),
		Type:             d.Get("type").(string),
	})
	if err != nil {
		return diag.Errorf("the network adapter could not be created: %s", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	setIdFromActivityConcernedItems(d, activity, "network_adapter")
	if err != nil {
		return diag.Errorf("failed to create network adapter, %s", err)
	}

	// We have to use update to set mac_type
	return computeNetworkAdapterUpdate(ctx, d, meta)
}

func computeNetworkAdapterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	// Récupérer l'adaptateur réseau par son ID
	networkAdapter, err := c.Compute().NetworkAdapter().Read(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if networkAdapter == nil {
		d.SetId("") // L'adaptateur n'existe plus, marquer la ressource comme supprimée
		return nil
	}

	// Mapper les données en utilisant la fonction helper
	adapterData := helpers.FlattenNetworkAdapter(networkAdapter)

	// Définir les données dans le state
	for k, v := range adapterData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func computeNetworkAdapterUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	activityId, err := c.Compute().NetworkAdapter().Update(ctx, &client.UpdateNetworkAdapterRequest{
		ID:           d.Id(),
		NewNetworkId: d.Get("network_id").(string),
		AutoConnect:  d.Get("auto_connect").(bool),
		MacAddress:   d.Get("mac_address").(string),
	})
	if err != nil {
		return diag.Errorf("failed to update network adapter, %s", err)
	}

	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("failed to update network adapter, %s", err)
	}

	if d.HasChange("connected") {
		var msg string
		var action func(context.Context, string) (string, error)
		if d.Get("connected").(bool) {
			msg = "connect"
			action = c.Compute().NetworkAdapter().Connect
		} else {
			msg = "disconnect"
			action = c.Compute().NetworkAdapter().Disconnect
		}

		// Connecting a network adapter can fail right after the VM has been powered
		// on so we retry here until we reach the timeout
		b := retry.NewFibonacci(1 * time.Second)
		b = retry.WithCappedDuration(20*time.Second, b)

		err = retry.Do(ctx, b, func(ctx context.Context) error {
			activityId, err = action(ctx, d.Id())
			if err != nil {
				return err
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			return err
		})
		if err != nil {
			return diag.Errorf("failed to %s network adapter, %s", msg, err)
		}
	}

	return computeNetworkAdapterRead(ctx, d, meta)
}

func computeNetworkAdapterDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().NetworkAdapter().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete network adapter: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete network adapter, %s", err)
	}
	return nil
}
