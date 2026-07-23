package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// publicCloudVMStorageTypeSkuSchema returns the Computed, read-only `sku` block
// exposing the priced SKU of a storage type. It is shared verbatim by the
// single and list datasources so the nested shape cannot drift between them.
// TypeFloat is the SDK type for the JSON `price` number (the first TypeFloat in
// the provider). The block is absent (empty list) when the API returns no sku.
func publicCloudVMStorageTypeSkuSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The priced SKU of the storage resource. Empty when the API returns no SKU for this storage type.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The SKU identifier (e.g. `csp:fr1:iaas:storage:bloc:medium:v1`).",
				},
				"price": {
					Type:        schema.TypeFloat,
					Computed:    true,
					Description: "The unit price of the SKU, expressed for `unit`.",
				},
				"unit": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The billing unit the price applies to (e.g. `1 Gio`).",
				},
				"description": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The human-readable description of the SKU (French).",
				},
				"description_en": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The human-readable description of the SKU (English).",
				},
			},
		},
	}
}

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
			"sku": publicCloudVMStorageTypeSkuSchema(),
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
		// Names are not guaranteed unique: refuse an ambiguous match rather
		// than silently picking one.
		var matches []string
		for _, s := range storageTypes {
			if s.Name == name {
				st = s
				matches = append(matches, s.ID)
			}
		}
		if st == nil {
			return diag.FromErr(fmt.Errorf("failed to find storage type named %q", name))
		}
		if len(matches) > 1 {
			return diag.FromErr(fmt.Errorf("found %d storage types named %q (ids: %s); use id to disambiguate", len(matches), name, strings.Join(matches, ", ")))
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
