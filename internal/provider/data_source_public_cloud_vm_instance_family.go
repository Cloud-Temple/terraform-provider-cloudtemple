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

func dataSourcePublicCloudVMInstanceFamily() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a Public Cloud VM Instances instance family, by `id` or by `name`.",

		ReadContext: publicCloudVMInstanceFamilyRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the instance family to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the instance family to retrieve (e.g. `General Purpose`). Conflicts with `id`.",
			},

			// Out
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
	}
}

// publicCloudVMSkusSchema is the shared, read-only schema of the priced SKU
// catalogue (vCPU and RAM) exposed by both the single and list instance-family
// datasources. Defining it once keeps the two datasources from drifting apart
// (#506). It returns a fresh *schema.Schema on each call so each datasource owns
// its own instance.
func publicCloudVMSkusSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The priced billing SKUs (vCPU and RAM) of the instance family.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The SKU identifier (e.g. `csp:fr1:vminstance:gp:vcpu:v1`).",
				},
				"price": {
					Type:        schema.TypeFloat,
					Computed:    true,
					Description: "The unit price of the SKU.",
				},
				"unit": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The billing unit of the SKU (e.g. `vcpu`, `gio`).",
				},
				"description": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The human-readable description of the SKU (French).",
				},
				"description_en": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The English description of the SKU.",
				},
			},
		},
	}
}

func publicCloudVMInstanceFamilyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var family *client.PublicCloudVMInstanceFamily

	if name := d.Get("name").(string); name != "" {
		families, err := c.PublicCloudVM().InstanceFamily().List(ctx)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find instance family named %q: %s", name, err))
		}
		// Names are not guaranteed unique: refuse an ambiguous match rather
		// than silently picking one.
		var matches []string
		for _, f := range families {
			if f.Name == name {
				family = f
				matches = append(matches, f.ID)
			}
		}
		if family == nil {
			return diag.FromErr(fmt.Errorf("failed to find instance family named %q", name))
		}
		if len(matches) > 1 {
			return diag.FromErr(fmt.Errorf("found %d instance families named %q (ids: %s); use id to disambiguate", len(matches), name, strings.Join(matches, ", ")))
		}
	} else {
		id := d.Get("id").(string)
		if id == "" {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
		var err error
		family, err = c.PublicCloudVM().InstanceFamily().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if family == nil {
			return diag.FromErr(fmt.Errorf("failed to find instance family with id %q", id))
		}
	}

	d.SetId(family.ID)
	for k, v := range helpers.FlattenPublicCloudVMInstanceFamily(family) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
