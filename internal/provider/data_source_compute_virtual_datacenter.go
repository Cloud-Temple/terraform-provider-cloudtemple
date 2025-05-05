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

func dataSourceVirtualDatacenter() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific virtual datacenter from a vCenter infrastructure.",

		ReadContext: computeVirtualDatacenterRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the virtual datacenter to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				RequiredWith:  []string{"machine_manager_id"},
				Description:   "The name of the virtual datacenter to retrieve. Conflicts with `id`. Requires `machine_manager_id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				RequiredWith:  []string{"name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the machine manager (vCenter) where the virtual datacenter is located. Required when using `name`.",
			},

			// Out
			"tenant_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the tenant that owns this virtual datacenter.",
			},
		},
	}
}

// computeVirtualDatacenterRead lit un datacenter virtuel et le mappe dans le state Terraform
func computeVirtualDatacenterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var datacenter *client.VirtualDatacenter
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		datacenters, err := c.Compute().VirtualDatacenter().List(ctx, &client.VirtualDatacenterFilter{
			Name:             name,
			MachineManagerId: d.Get("machine_manager_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual datacenter named %q: %s", name, err))
		}
		for _, dc := range datacenters {
			if dc.Name == name {
				datacenter = dc
				break
			}
		}
		if datacenter == nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual datacenter named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			datacenter, err = c.Compute().VirtualDatacenter().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if datacenter == nil {
				return diag.FromErr(fmt.Errorf("failed to find virtual datacenter with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(datacenter.ID)

	// Mapper les données en utilisant la fonction helper
	datacenterData := helpers.FlattenVirtualDatacenter(datacenter)

	// Définir les données dans le state
	for k, v := range datacenterData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
