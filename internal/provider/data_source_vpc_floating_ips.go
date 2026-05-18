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
		Description: "Used to retrieve a list of floating IPs. Can be filtered by VPC ID.",

		ReadContext: dataSourceVPCFloatingIPsRead,

		Schema: map[string]*schema.Schema{
			// In
			"vpc_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter floating IPs by VPC ID. If not provided, returns all floating IPs for the tenant.",
			},

			// Out
			"floating_ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of floating IPs matching the filter criteria.",

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
				},
			},
		},
	}
}

func dataSourceVPCFloatingIPsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	vpcID := d.Get("vpc_id").(string)

	floatingIPs, err := c.VPC().FloatingIP().List(ctx, &client.FloatingIPFilter{
		VpcID: vpcID,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("floating_ips")

	// Map data using the helper function
	tfFloatingIPs := make([]map[string]interface{}, len(floatingIPs))
	for i, floatingIP := range floatingIPs {
		tfFloatingIPs[i] = helpers.FlattenFloatingIP(floatingIP)
	}

	if err := d.Set("floating_ips", tfFloatingIPs); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
