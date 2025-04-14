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

func dataSourceOpenIaasMachineManager() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve an Availability Zone.",

		ReadContext: computeOpenIaaSMachineManagerRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},

			// Out
			"os_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"os_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"xoa_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// computeOpenIaaSMachineManagerRead lit un gestionnaire de machines OpenIaaS et le mappe dans le state Terraform
func computeOpenIaaSMachineManagerRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var machineManager *client.OpenIaaSMachineManager
	var err error

	// Recherche par ID
	id := d.Get("id").(string)
	if id != "" {
		machineManager, err = c.Compute().OpenIaaS().MachineManager().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if machineManager == nil {
			return diag.FromErr(fmt.Errorf("failed to find machine manager with id %q", id))
		}
	} else {
		// Recherche par nom
		name := d.Get("name").(string)
		if name != "" {
			machineManagers, err := c.Compute().OpenIaaS().MachineManager().List(ctx)
			if err != nil {
				return diag.FromErr(fmt.Errorf("failed to list machine managers: %s", err))
			}
			for _, mm := range machineManagers {
				if mm.Name == name {
					machineManager = mm
					break
				}
			}
			if machineManager == nil {
				return diag.FromErr(fmt.Errorf("failed to find machine manager named %q", name))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(machineManager.ID)

	// Mapper les données en utilisant la fonction helper
	machineManagerData := helpers.FlattenOpenIaaSMachineManager(machineManager)

	// Définir les données dans le state
	for k, v := range machineManagerData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
