package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMStorageTypes() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all Public Cloud VM Instances storage types available to the tenant, optionally filtered by an availability-zone/instance-family pair.",

		ReadContext: publicCloudVMStorageTypesRead,

		Schema: map[string]*schema.Schema{
			// In
			//
			// The two filters are optional but PAIRED (RequiredWith both ways):
			// the storage-types API rejects a request carrying only one of them.
			"availability_zone_id": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"instance_family_id"},
				ValidateFunc: validation.IsUUID,
				Description:  "Filter storage types by availability zone ID. Must be set together with `instance_family_id`.",
			},
			"instance_family_id": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"availability_zone_id"},
				ValidateFunc: validation.IsUUID,
				Description:  "Filter storage types by instance family ID. Must be set together with `availability_zone_id`.",
			},

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
						"min_size_gib": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The minimum disk size (GiB) allowed for this storage type.",
						},
						"max_size_gib": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The maximum disk size (GiB) allowed for this storage type.",
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

	azID := d.Get("availability_zone_id").(string)
	familyID := d.Get("instance_family_id").(string)
	if azID != "" && familyID != "" {
		if err := validatePublicCloudVMStorageTypeFilterPair(ctx, c, azID, familyID); err != nil {
			return diag.FromErr(err)
		}
	}

	storageTypes, err := c.PublicCloudVM().StorageType().List(ctx, &client.PublicCloudVMStorageTypeFilter{
		AvailabilityZoneID: azID,
		InstanceFamilyID:   familyID,
	})
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
