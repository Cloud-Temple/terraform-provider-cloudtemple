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
		Description: "Used to retrieve a specific VPC floating IP.",

		ReadContext: dataSourceVPCFloatingIPRead,

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
				Description: "The ID of the static IP this floating IP is bound to, if any.",
			},
			"static_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The address of the static IP this floating IP is bound to, if any.",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the VPC this floating IP is associated with, if any.",
			},
			"private_network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the private network this floating IP is associated with, if any.",
			},
		},
	}
}

func dataSourceVPCFloatingIPRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	id := d.Get("id").(string)

	floatingIP, err := c.VPC().FloatingIP().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if floatingIP == nil {
		return diag.FromErr(fmt.Errorf("failed to find floating IP with id %q", id))
	}

	d.SetId(floatingIP.ID)

	for k, v := range helpers.FlattenFloatingIP(floatingIP) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
