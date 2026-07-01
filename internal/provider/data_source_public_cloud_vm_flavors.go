package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMFlavors() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all Public Cloud VM Instances flavors of the tenant, optionally filtered by instance family.",

		ReadContext: publicCloudVMFlavorsRead,

		Schema: map[string]*schema.Schema{
			// In
			"instance_family_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter flavors by instance family ID.",
			},

			// Out
			"flavors": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of flavors.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the flavor.",
						},
						"instance_family_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the instance family this flavor belongs to.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the flavor (e.g. `dev-micro`).",
						},
						"vcpu": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of vCPUs of the flavor.",
						},
						"ram_gb": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The amount of RAM of the flavor, in GB.",
						},
					},
				},
			},
		},
	}
}

func publicCloudVMFlavorsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	flavors, err := c.PublicCloudVM().Flavor().List(ctx, &client.PublicCloudVMFlavorFilter{
		FamilyID: d.Get("instance_family_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_flavors")

	tfFlavors := make([]map[string]interface{}, len(flavors))
	for i, f := range flavors {
		tfFlavors[i] = helpers.FlattenPublicCloudVMFlavor(f)
	}

	if err := d.Set("flavors", tfFlavors); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
