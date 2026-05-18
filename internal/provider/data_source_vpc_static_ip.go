package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVPCStaticIP() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific static IP.",

		ReadContext: vpcStaticIPRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the static IP to retrieve.",
			},

			// Out
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The static IP address.",
			},
			"mac_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The MAC address of the network adapter.",
			},
			"virtual_machine_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the virtual machine associated with this static IP.",
			},
			"network_adapter_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the network adapter associated with this static IP.",
			},
			"source": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The source of the virtual machine (xoa, vmware, or custom).",
			},
			"resource_description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the resource (for custom sources).",
			},
			"floating_ip_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the floating IP bound to this static IP.",
			},
			"floating_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The address of the floating IP bound to this static IP.",
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

func vpcStaticIPRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	id := d.Get("id").(string)

	staticIP, err := c.VPC().StaticIP().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if staticIP == nil {
		return diag.FromErr(fmt.Errorf("failed to find static IP with id %q", id))
	}

	d.SetId(staticIP.ID)

	// Map data using the helper function
	staticIPData := helpers.FlattenStaticIP(staticIP)

	// Set data in state
	for k, v := range staticIPData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
