package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMStorageType() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a single Public Cloud VM Instances storage type, by `id` or by `name`. The API has no by-id endpoint for storage types; selection is done by listing and matching.",

		ReadContext: publicCloudVMStorageTypeRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the storage type to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the storage type to retrieve (e.g. `Standard`). Conflicts with `id`.",
			},

			// Out
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The human-readable description of the storage type.",
			},
			"iops_hint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "An indicative IOPS hint for the storage type (e.g. `~1500 IOPS/TB`).",
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
		},
	}
}

func publicCloudVMStorageTypeRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	storageTypes, err := c.PublicCloudVM().StorageType().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	var st *client.PublicCloudVMStorageType
	if name := d.Get("name").(string); name != "" {
		for _, s := range storageTypes {
			if s.Name == name {
				st = s
				break
			}
		}
		if st == nil {
			return diag.FromErr(fmt.Errorf("failed to find storage type named %q", name))
		}
	} else {
		id := d.Get("id").(string)
		if id == "" {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
		for _, s := range storageTypes {
			if s.ID == id {
				st = s
				break
			}
		}
		if st == nil {
			return diag.FromErr(fmt.Errorf("failed to find storage type with id %q", id))
		}
	}

	d.SetId(st.ID)
	for k, v := range helpers.FlattenPublicCloudVMStorageType(st) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
