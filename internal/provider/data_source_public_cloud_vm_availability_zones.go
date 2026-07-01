package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMAvailabilityZones() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all Public Cloud VM Instances availability zones of the tenant.",

		ReadContext: publicCloudVMAvailabilityZonesRead,

		Schema: map[string]*schema.Schema{
			// In
			"region_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter availability zones by region ID.",
			},

			// Out
			"availability_zones": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of availability zones.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the availability zone.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the availability zone (e.g. `fr1-az01`).",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The human-readable description of the availability zone.",
						},
						"region_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the region this availability zone belongs to.",
						},
						"is_enabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the availability zone is enabled for the tenant.",
						},
						"compatible_families": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The instance families available in this availability zone.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the instance family.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the instance family.",
									},
								},
							},
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The creation date of the availability zone (RFC3339).",
						},
						"updated_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The last update date of the availability zone (RFC3339).",
						},
					},
				},
			},
		},
	}
}

func publicCloudVMAvailabilityZonesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	zones, err := c.PublicCloudVM().AvailabilityZone().List(ctx, &client.PublicCloudVMAvailabilityZoneFilter{
		RegionID: d.Get("region_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_availability_zones")

	tfZones := make([]map[string]interface{}, len(zones))
	for i, z := range zones {
		tfZones[i] = helpers.FlattenPublicCloudVMAvailabilityZone(z)
	}

	if err := d.Set("availability_zones", tfZones); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
