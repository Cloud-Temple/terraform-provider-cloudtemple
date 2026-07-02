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

func dataSourcePublicCloudVMFlavor() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a single Public Cloud VM Instances flavor (a predefined vCPU/RAM sizing pair), by `id` or by `name`. The API has no by-id endpoint for flavors; selection is done by listing and matching.",

		ReadContext: publicCloudVMFlavorRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the flavor to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the flavor to retrieve (e.g. `dev-micro`). Conflicts with `id`.",
			},
			"instance_family_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the instance family, used to narrow the search. Populated with the flavor's family on read.",
			},

			// Out
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
	}
}

func publicCloudVMFlavorRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	flavors, err := c.PublicCloudVM().Flavor().List(ctx, &client.PublicCloudVMFlavorFilter{
		FamilyID: d.Get("instance_family_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	var flavor *client.PublicCloudVMFlavor
	if name := d.Get("name").(string); name != "" {
		// Names are not guaranteed unique: refuse an ambiguous match rather
		// than silently picking one.
		var matches []string
		for _, f := range flavors {
			if f.Name == name {
				flavor = f
				matches = append(matches, f.ID)
			}
		}
		if flavor == nil {
			return diag.FromErr(fmt.Errorf("failed to find flavor named %q", name))
		}
		if len(matches) > 1 {
			return diag.FromErr(fmt.Errorf("found %d flavors named %q (ids: %s); narrow with instance_family_id or use id", len(matches), name, strings.Join(matches, ", ")))
		}
	} else {
		id := d.Get("id").(string)
		if id == "" {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
		for _, f := range flavors {
			if f.ID == id {
				flavor = f
				break
			}
		}
		if flavor == nil {
			return diag.FromErr(fmt.Errorf("failed to find flavor with id %q", id))
		}
	}

	d.SetId(flavor.ID)
	for k, v := range helpers.FlattenPublicCloudVMFlavor(flavor) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
