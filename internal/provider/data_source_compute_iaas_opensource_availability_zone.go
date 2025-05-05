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
		Description: "Used to retrieve a specific machine manager from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSMachineManagerRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the machine manager to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the machine manager to retrieve. Conflicts with `id`.",
			},

			// Out
			"os_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The operating system version of the machine manager.",
			},
			"os_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The operating system name of the machine manager.",
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
			return diag.FromErr(fmt.Errorf("failed to find availability zone with id %q", id))
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
				return diag.FromErr(fmt.Errorf("failed to find availability zone named %q", name))
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
