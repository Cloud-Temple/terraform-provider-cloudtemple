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

func dataSourcePublicCloudVMAvailabilityZone() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a Public Cloud VM Instances availability zone, by `id` or by `name`.",

		ReadContext: publicCloudVMAvailabilityZoneRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the availability zone to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the availability zone to retrieve (e.g. `fr1-az01`). Conflicts with `id`.",
			},
			"region_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				Description:   "The ID of the region, used to narrow the search when looking up by `name`. Populated with the zone's region on read.",
			},

			// Out
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The human-readable description of the availability zone.",
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
	}
}

func publicCloudVMAvailabilityZoneRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var az *client.PublicCloudVMAvailabilityZone

	if name := d.Get("name").(string); name != "" {
		zones, err := c.PublicCloudVM().AvailabilityZone().List(ctx, &client.PublicCloudVMAvailabilityZoneFilter{
			RegionID: d.Get("region_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find availability zone named %q: %s", name, err))
		}
		for _, z := range zones {
			if z.Name == name {
				az = z
				break
			}
		}
		if az == nil {
			return diag.FromErr(fmt.Errorf("failed to find availability zone named %q", name))
		}
	} else {
		id := d.Get("id").(string)
		if id == "" {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
		var err error
		az, err = c.PublicCloudVM().AvailabilityZone().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if az == nil {
			return diag.FromErr(fmt.Errorf("failed to find availability zone with id %q", id))
		}
	}

	d.SetId(az.ID)
	for k, v := range helpers.FlattenPublicCloudVMAvailabilityZone(az) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
