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

func dataSourceNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific network adapter attached to a virtual machine.",

		ReadContext: dataSourceNetworkAdapterRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the network adapter to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"virtual_machine_id"},
				Description:   "The name of the network adapter to retrieve. Requires `virtual_machine_id`. Conflicts with `id`.",
			},
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				RequiredWith:  []string{"name"},
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the virtual machine the network adapter is attached to. Required when searching by `name`.",
			},

			// Out
			"network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the network this adapter is connected to.",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the network adapter (e.g., VMXNET3, E1000).",
			},
			"mac_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of MAC address assignment (e.g., MANUAL, GENERATED).",
			},
			"mac_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The MAC address of the network adapter.",
			},
			"connected": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the network adapter is currently connected.",
			},
			"auto_connect": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the network adapter is configured to connect automatically when the virtual machine powers on.",
			},
		},
	}
}

// dataSourceNetworkAdapterRead lit un adaptateur réseau et le mappe dans le state Terraform
func dataSourceNetworkAdapterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var adapter *client.NetworkAdapter
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		virtualMachineId := d.Get("virtual_machine_id").(string)
		if virtualMachineId == "" {
			return diag.FromErr(fmt.Errorf("virtual_machine_id is required when searching by name"))
		}

		adapters, err := c.Compute().NetworkAdapter().List(ctx, &client.NetworkAdapterFilter{
			VirtualMachineID: virtualMachineId,
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find network adapter named %q: %s", name, err))
		}
		for _, a := range adapters {
			if a.Name == name {
				adapter = a
				break
			}
		}
		if adapter == nil {
			return diag.FromErr(fmt.Errorf("failed to find network adapter named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			adapter, err = c.Compute().NetworkAdapter().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if adapter == nil {
				return diag.FromErr(fmt.Errorf("failed to find network adapter with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(adapter.ID)

	// Mapper les données en utilisant la fonction helper
	adapterData := helpers.FlattenNetworkAdapter(adapter)

	// Définir les données dans le state
	for k, v := range adapterData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
