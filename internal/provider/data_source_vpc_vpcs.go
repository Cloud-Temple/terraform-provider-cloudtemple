package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVPCVPCs() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all VPCs.",

		ReadContext: dataSourceVPCVPCsRead,

		Schema: map[string]*schema.Schema{
			// Out
			"vpcs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of VPCs.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the VPC.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the VPC.",
						},
						"internet_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internet IP address assigned to the VPC, if any.",
						},
						"private_network_count": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of private networks in this VPC.",
						},
						"static_ip_count": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of static IPs in this VPC.",
						},
						"floating_ip_count": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of floating IPs in this VPC.",
						},
					},
				},
			},
		},
	}
}

func dataSourceVPCVPCsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	vpcs, err := c.VPC().VPC().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("vpcs")

	tfVPCs := make([]map[string]interface{}, len(vpcs))
	for i, vpc := range vpcs {
		tfVPCs[i] = helpers.FlattenVPC(vpc)
	}

	if err := d.Set("vpcs", tfVPCs); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
