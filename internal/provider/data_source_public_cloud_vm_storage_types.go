package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePublicCloudVMStorageTypes() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all Public Cloud VM Instances storage types available to the tenant.",

		ReadContext: publicCloudVMStorageTypesRead,

		Schema: map[string]*schema.Schema{
			// Out
			"storage_types": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of storage types.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the storage type.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the storage type (e.g. `Standard`).",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The human-readable description of the storage type.",
						},
						"iops_hint": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "An indicative IOPS hint for the storage type.",
						},
						"min_size_gb": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The minimum disk size (GB) allowed for this storage type.",
						},
						"max_size_gb": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The maximum disk size (GB) allowed for this storage type.",
						},
						"is_available": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the storage type is currently available.",
						},
						"sku": publicCloudVMStorageTypeSkuSchema(),
					},
				},
			},
		},
	}
}

func publicCloudVMStorageTypesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	storageTypes, err := c.PublicCloudVM().StorageType().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_storage_types")

	tfStorageTypes := make([]map[string]interface{}, len(storageTypes))
	for i, st := range storageTypes {
		tfStorageTypes[i] = helpers.FlattenPublicCloudVMStorageType(st)
	}

	if err := d.Set("storage_types", tfStorageTypes); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
