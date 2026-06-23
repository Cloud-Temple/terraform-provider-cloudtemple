package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVPCPrivateNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific VPC private network.",

		ReadContext: dataSourceVPCPrivateNetworkRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the private network to retrieve.",
			},

			// Out
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the private network, if any.",
			},
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP network address in CIDR notation.",
			},
			"vlan_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The VLAN ID of the private network.",
			},
			"static_ip_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of static IPs configured for this private network.",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the VPC this private network belongs to.",
			},
		},
	}
}

func dataSourceVPCPrivateNetworkRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	id := d.Get("id").(string)

	network, err := c.VPC().PrivateNetwork().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if network == nil {
		return diag.FromErr(fmt.Errorf("failed to find private network with id %q", id))
	}

	d.SetId(network.ID)

	for k, v := range helpers.FlattenPrivateNetwork(network) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
