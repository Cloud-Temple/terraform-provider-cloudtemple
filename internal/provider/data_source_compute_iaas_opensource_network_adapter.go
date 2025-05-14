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

func dataSourceOpenIaasNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific network adapter from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSNetworkAdapterRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the network adapter to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the network adapter to retrieve. Conflicts with `id`.",
			},
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the virtual machine the network adapter is attached to. Required when searching by `name`.",
			},

			// Out
			"machine_manager_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the machine manager this network adapter belongs to.",
			},
			"internal_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internal identifier of the network adapter in the Open IaaS system.",
			},
			"network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the network this adapter is connected to.",
			},
			"mac_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The MAC address of the network adapter.",
			},
			"mtu": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The Maximum Transmission Unit (MTU) size in bytes.",
			},
			"attached": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the network adapter is attached to a virtual machine.",
			},
		},
	}
}

// computeOpenIaaSNetworkAdapterRead lit un adaptateur réseau OpenIaaS et le mappe dans le state Terraform
func computeOpenIaaSNetworkAdapterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var adapter *client.OpenIaaSNetworkAdapter
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		virtualMachineId := d.Get("virtual_machine_id").(string)
		if virtualMachineId == "" {
			return diag.FromErr(fmt.Errorf("virtual_machine_id is required when searching by name"))
		}

		adapters, err := c.Compute().OpenIaaS().NetworkAdapter().List(ctx, &client.OpenIaaSNetworkAdapterFilter{
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
			adapter, err = c.Compute().OpenIaaS().NetworkAdapter().Read(ctx, id)
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
	adapterData := helpers.FlattenOpenIaaSNetworkAdapter(adapter)

	// Définir les données dans le state
	for k, v := range adapterData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
