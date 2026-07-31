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

// validatePublicCloudVMStorageTypeFilterPair fails closed when either id of the
// paired storage-type filters does not exist in its catalogue — and, when the
// availability zone advertises its compatible families, when the pair is not
// compatible. The storage-types API silently ignores unknown filter VALUES
// (live-probed: a syntactically valid but nonexistent instanceFamilyId returns
// the full catalogue with HTTP 200), so without this check a UUID typo would
// masquerade as a successfully filtered read.
func validatePublicCloudVMStorageTypeFilterPair(ctx context.Context, c *client.Client, azID, familyID string) error {
	az, err := c.PublicCloudVM().AvailabilityZone().Read(ctx, azID)
	if err != nil {
		return fmt.Errorf("failed to check availability zone %q: %w", azID, err)
	}
	if az == nil {
		return fmt.Errorf("availability zone %q does not exist: the storage-types API silently ignores unknown filter values, so the result would NOT actually be filtered", azID)
	}
	family, err := c.PublicCloudVM().InstanceFamily().Read(ctx, familyID)
	if err != nil {
		return fmt.Errorf("failed to check instance family %q: %w", familyID, err)
	}
	if family == nil {
		return fmt.Errorf("instance family %q does not exist: the storage-types API silently ignores unknown filter values, so the result would NOT actually be filtered", familyID)
	}
	// Compatibility is enforced only when the zone actually advertises its
	// compatible families: an empty list cannot prove an incompatibility, so it
	// must not fail-close valid pairs against an API that does not populate it.
	if len(az.CompatibleFamilies) > 0 {
		compatible := make([]string, 0, len(az.CompatibleFamilies))
		for _, f := range az.CompatibleFamilies {
			if f.ID == familyID {
				return nil
			}
			compatible = append(compatible, fmt.Sprintf("%s (%s)", f.ID, f.Name))
		}
		return fmt.Errorf("instance family %q (%s) is not advertised as compatible with availability zone %q (%s); compatible families: %s", familyID, family.Name, azID, az.Name, strings.Join(compatible, ", "))
	}
	return nil
}

// storageTypeFilterContext renders the filter pair for not-found errors, so a
// miss inside a narrowed catalogue is not mistaken for a global absence.
func storageTypeFilterContext(azID, familyID string) string {
	if azID == "" && familyID == "" {
		return ""
	}
	return fmt.Sprintf(" within the catalogue filtered by availability_zone_id=%q, instance_family_id=%q", azID, familyID)
}

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
		Description: "Used to retrieve a single Public Cloud VM Instances storage type, by `id` or by `name`. The API has no by-id endpoint for storage types; selection is done by listing and matching. The optional paired filters narrow the catalogue in which `id`/`name` is resolved.",

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
			// The two filters are optional but PAIRED (RequiredWith both ways):
			// the storage-types API rejects a request carrying only one of them.
			"availability_zone_id": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"instance_family_id"},
				ValidateFunc: validation.IsUUID,
				Description:  "Availability zone ID narrowing the catalogue in which the storage type is looked up. Must be set together with `instance_family_id`.",
			},
			"instance_family_id": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"availability_zone_id"},
				ValidateFunc: validation.IsUUID,
				Description:  "Instance family ID narrowing the catalogue in which the storage type is looked up. Must be set together with `availability_zone_id`.",
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
			return diag.FromErr(fmt.Errorf("failed to find storage type named %q%s", name, storageTypeFilterContext(azID, familyID)))
		}
		if len(matches) > 1 {
			return diag.FromErr(fmt.Errorf("found %d storage types named %q (ids: %s); narrow with availability_zone_id and instance_family_id, or use id", len(matches), name, strings.Join(matches, ", ")))
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
			return diag.FromErr(fmt.Errorf("failed to find storage type with id %q%s", id, storageTypeFilterContext(azID, familyID)))
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
