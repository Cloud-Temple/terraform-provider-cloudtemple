package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVPCPrivateNetworks() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all VPC private networks. Can be filtered by VPC ID.",

		ReadContext: dataSourceVPCPrivateNetworksRead,

		Schema: map[string]*schema.Schema{
			// In
			"vpc_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter private networks by VPC ID. If not provided, returns all private networks for the tenant.",
			},

			// Out
			"private_networks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of private networks matching the filter.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the private network.",
						},
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
				},
			},
		},
	}
}

func dataSourceVPCPrivateNetworksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	vpcID := d.Get("vpc_id").(string)

	networks, err := c.VPC().PrivateNetwork().List(ctx, &client.PrivateNetworkFilter{
		VpcID: vpcID,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("private_networks")

	tfNetworks := make([]map[string]interface{}, len(networks))
	for i, network := range networks {
		tfNetworks[i] = helpers.FlattenPrivateNetwork(network)
	}

	if err := d.Set("private_networks", tfNetworks); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
