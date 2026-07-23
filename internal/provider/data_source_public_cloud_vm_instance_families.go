package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePublicCloudVMInstanceFamilies() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all Public Cloud VM Instances instance families of the tenant.",

		ReadContext: publicCloudVMInstanceFamiliesRead,

		Schema: map[string]*schema.Schema{
			// Out
			"instance_families": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of instance families.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the instance family.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the instance family (e.g. `General Purpose`).",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The human-readable description of the instance family.",
						},
						"vcpu_min": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The minimum number of vCPUs allowed in this family.",
						},
						"vcpu_max": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The maximum number of vCPUs allowed in this family.",
						},
						"ram_min_gb": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The minimum amount of RAM (GB) allowed in this family.",
						},
						"ram_max_gb": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The maximum amount of RAM (GB) allowed in this family.",
						},
						"skus": publicCloudVMSkusSchema(),
					},
				},
			},
		},
	}
}

func publicCloudVMInstanceFamiliesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	families, err := c.PublicCloudVM().InstanceFamily().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_instance_families")

	tfFamilies := make([]map[string]interface{}, len(families))
	for i, f := range families {
		tfFamilies[i] = helpers.FlattenPublicCloudVMInstanceFamily(f)
	}

	if err := d.Set("instance_families", tfFamilies); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
