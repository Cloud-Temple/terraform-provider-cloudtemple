package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVPCFloatingIP() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a VPC floating IP. A floating IP is an IP address that can be dynamically associated with a static IP within the same VPC.",

		CreateContext: resourceVPCFloatingIPCreate,
		ReadContext:   resourceVPCFloatingIPRead,
		UpdateContext: resourceVPCFloatingIPUpdate,
		DeleteContext: resourceVPCFloatingIPDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The description of the floating IP.",
			},
			"static_ip_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The ID of the static IP to bind to this floating IP.",
			},

			// Out
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the floating IP.",
			},
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address of the floating IP.",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the VPC this floating IP belongs to.",
			},
			"private_network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the private network this floating IP is associated with.",
			},
		},
	}
}

func resourceVPCFloatingIPCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Create request with count always set to 1
	req := &client.CreateFloatingIPRequest{
		Count: 1,
	}

	// Create the floating IP
	activityId, err := c.VPC().FloatingIP().Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}

	// Wait for the activity to complete
	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("error waiting for floating IP creation: %s", err)
	}

	// Extract the floating IP ID from the activity result
	setIdFromActivityState(d, activity)
	if d.Id() == "" {
		return diag.Errorf("failed to get floating IP ID from activity")
	}

	// Bind to static IP if specified
	if staticIPID, ok := d.GetOk("static_ip_id"); ok && staticIPID.(string) != "" {
		activityId, err := c.VPC().FloatingIP().Bind(ctx, d.Id(), staticIPID.(string))
		if err != nil {
			return diag.Errorf("error binding floating IP to static IP: %s", err)
		}

		// Wait for the bind activity to complete
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("error waiting for floating IP bind: %s", err)
		}
	}

	// Read the floating IP to populate all fields
	return resourceVPCFloatingIPRead(ctx, d, meta)
}

func resourceVPCFloatingIPRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	floatingIP, err := c.VPC().FloatingIP().Read(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if floatingIP == nil {
		d.SetId("")
		return nil
	}

	// Set the fields
	if err := d.Set("ip_address", floatingIP.IPAddress); err != nil {
		return diag.FromErr(err)
	}

	// Only set description if it's not already set by the user
	// This prevents overwriting user input with empty values from the API
	if _, ok := d.GetOk("description"); !ok || floatingIP.Description != "" {
		if err := d.Set("description", floatingIP.Description); err != nil {
			return diag.FromErr(err)
		}
	}

	// Set static IP ID if present
	if floatingIP.StaticIP != nil {
		if err := d.Set("static_ip_id", floatingIP.StaticIP.ID); err != nil {
			return diag.FromErr(err)
		}
	}

	// Set VPC ID if present
	if floatingIP.VPC != nil {
		if err := d.Set("vpc_id", floatingIP.VPC.ID); err != nil {
			return diag.FromErr(err)
		}
	}

	// Set Private Network ID if present
	if floatingIP.PrivateNetwork != nil {
		if err := d.Set("private_network_id", floatingIP.PrivateNetwork.ID); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceVPCFloatingIPUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	if d.HasChange("description") {
		req := &client.UpdateFloatingIPRequest{
			ID:          d.Id(),
			Description: d.Get("description").(string),
		}

		// Update the floating IP
		activityId, err := c.VPC().FloatingIP().Update(ctx, req)
		if err != nil {
			return diag.FromErr(err)
		}

		// Wait for the activity to complete
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("error waiting for floating IP update: %s", err)
		}
	}

	if d.HasChange("static_ip_id") {
		oldValue, newValue := d.GetChange("static_ip_id")
		oldStaticIPID := oldValue.(string)
		newStaticIPID := newValue.(string)

		// Unbind from old static IP if it exists
		if oldStaticIPID != "" {
			activityId, err := c.VPC().FloatingIP().Unbind(ctx, d.Id(), oldStaticIPID)
			if err != nil {
				return diag.Errorf("error unbinding floating IP from static IP: %s", err)
			}

			// Wait for the unbind activity to complete
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("error waiting for floating IP unbind: %s", err)
			}
		}

		// Bind to new static IP if it exists
		if newStaticIPID != "" {
			activityId, err := c.VPC().FloatingIP().Bind(ctx, d.Id(), newStaticIPID)
			if err != nil {
				return diag.Errorf("error binding floating IP to static IP: %s", err)
			}

			// Wait for the bind activity to complete
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("error waiting for floating IP bind: %s", err)
			}
		}
	}

	// Read the floating IP to get the updated state
	return resourceVPCFloatingIPRead(ctx, d, meta)
}

func resourceVPCFloatingIPDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Delete the floating IP
	activityId, err := c.VPC().FloatingIP().Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Wait for the activity to complete
	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("error waiting for floating IP deletion: %s", err)
	}

	return nil
}
