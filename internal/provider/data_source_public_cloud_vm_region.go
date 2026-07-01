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

func dataSourcePublicCloudVMRegion() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a Public Cloud VM Instances region, by `id` or by `name`.",

		ReadContext: publicCloudVMRegionRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the region to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the region to retrieve (e.g. `fr1`). Conflicts with `id`.",
			},

			// Out
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
	}
}

func publicCloudVMRegionRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var region *client.PublicCloudVMRegion

	if name := d.Get("name").(string); name != "" {
		// No server-side name filter on /regions: list and match by exact name.
		regions, err := c.PublicCloudVM().Region().List(ctx)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find region named %q: %s", name, err))
		}
		for _, r := range regions {
			if r.Name == name {
				region = r
				break
			}
		}
		if region == nil {
			return diag.FromErr(fmt.Errorf("failed to find region named %q", name))
		}
	} else {
		id := d.Get("id").(string)
		if id == "" {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
		var err error
		region, err = c.PublicCloudVM().Region().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if region == nil {
			return diag.FromErr(fmt.Errorf("failed to find region with id %q", id))
		}
	}

	d.SetId(region.ID)
	for k, v := range helpers.FlattenPublicCloudVMRegion(region) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
