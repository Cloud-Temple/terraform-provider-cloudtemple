package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVPCFloatingIP() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific floating IP.",

		ReadContext: vpcFloatingIPRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the floating IP to retrieve.",
			},

			// Out
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The floating IP address.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the floating IP.",
			},
			"static_ip_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the static IP associated with this floating IP.",
			},
			"static_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The address of the static IP associated with this floating IP.",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the VPC.",
			},
			"private_network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the private network.",
			},
		},
	}
}

func vpcFloatingIPRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	id := d.Get("id").(string)

	floatingIP, err := c.VPC().FloatingIP().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if floatingIP == nil {
		return diag.FromErr(fmt.Errorf("failed to find floating IP with id %q", id))
	}

	d.SetId(floatingIP.ID)

	// Map data using the helper function
	floatingIPData := helpers.FlattenFloatingIP(floatingIP)

	// Set data in state
	for k, v := range floatingIPData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
