package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
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
				Description: "The MAC address of the network adapter.",
				Optional:    true,
				Computed:    true,
			},
			"attached": {
				Type:        schema.TypeBool,
				Description: "Whether the network adapter is attached.",
				Optional:    true,
				Default:     true,
			},

			// Out
			"id": {
				Type:        schema.TypeString,
				Description: "The ID of the network adapter.",
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

	activityId, err := c.Compute().OpenIaaS().NetworkAdapter().Create(ctx, &client.CreateOpenIaasNetworkAdapterRequest{
		VirtualMachineID: d.Get("virtual_machine_id").(string),
		NetworkID:        d.Get("network_id").(string),
		MAC:              d.Get("mac_address").(string),
	})
	if err != nil {
		return diag.Errorf("the network adapter could not be created: %s", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions((ctx)))
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

	networkAdapter, err := c.Compute().OpenIaaS().NetworkAdapter().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("the network adapter could not be read: %s", err)
	}
	if networkAdapter == nil {
		return diag.Errorf("the network adapter was not found")
	}

	sw := newStateWriter(d)
	// Set the retrieved data to the schema
	sw.set("name", networkAdapter.Name)
	sw.set("machine_manager_id", networkAdapter.MachineManagerID) // DevNote : This will change in future API versions
	sw.set("mtu", networkAdapter.MTU)
	sw.set("attached", networkAdapter.Attached)
	sw.set("mac_address", networkAdapter.MacAddress)
	sw.set("network_id", networkAdapter.Network.ID)
	sw.set("virtual_machine_id", networkAdapter.VirtualMachineID)

	return sw.diags
}

func openIaasNetworkAdapterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)

	if d.HasChange("network_id") || d.HasChange("mac_address") {
		activityId, err := c.Compute().OpenIaaS().NetworkAdapter().Update(ctx, d.Id(), &client.UpdateOpenIaasNetworkAdapterRequest{
			MAC:       d.Get("mac_address").(string),
			NetworkID: d.Get("network_id").(string),
		})
		if err != nil {
			return diag.Errorf("the network adapter could not be updated: %s", err)
		}
		if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
			return diag.Errorf("failed to update network adapter, %s", err)
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
