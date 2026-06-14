package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVPCFloatingIPs() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all VPC floating IPs. Can be filtered by VPC ID.",

		ReadContext: dataSourceVPCFloatingIPsRead,

		Schema: map[string]*schema.Schema{
			// In
			"vpc_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter floating IPs that have a static IP in the given VPC. If not provided, returns all floating IPs for the tenant.",
			},

			// Out
			"floating_ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of floating IPs matching the filter.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the floating IP.",
						},
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
				},
			},
		},
	}
}

func dataSourceVPCFloatingIPsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	vpcID := d.Get("vpc_id").(string)

	floatingIPs, err := c.VPC().FloatingIP().List(ctx, &client.FloatingIPFilter{
		VpcID: vpcID,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("floating_ips")

	tfFloatingIPs := make([]map[string]interface{}, len(floatingIPs))
	for i, floatingIP := range floatingIPs {
		tfFloatingIPs[i] = helpers.FlattenFloatingIP(floatingIP)
	}

	if err := d.Set("floating_ips", tfFloatingIPs); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
