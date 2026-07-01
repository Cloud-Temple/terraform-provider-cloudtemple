package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePublicCloudVMRegions() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all Public Cloud VM Instances regions of the tenant.",

		ReadContext: publicCloudVMRegionsRead,

		Schema: map[string]*schema.Schema{
			// Out
			"regions": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of regions.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the region.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the region (e.g. `fr1`).",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The human-readable description of the region.",
						},
						"country_code": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ISO 3166-1 alpha-2 country code of the region.",
						},
						"geography": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The geography the region belongs to (e.g. `Europe`).",
						},
						"is_enabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the region is enabled for the tenant.",
						},
						"az_count": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of availability zones in the region.",
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The creation date of the region (RFC3339).",
						},
						"updated_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The last update date of the region (RFC3339).",
						},
					},
				},
			},
		},
	}
}

func publicCloudVMRegionsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	regions, err := c.PublicCloudVM().Region().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_regions")

	tfRegions := make([]map[string]interface{}, len(regions))
	for i, region := range regions {
		tfRegions[i] = helpers.FlattenPublicCloudVMRegion(region)
	}

	if err := d.Set("regions", tfRegions); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
