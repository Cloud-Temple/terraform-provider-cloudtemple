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

func dataSourceOpenIaasNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific network from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSNetworkRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name", "machine_manager_id", "pool_id"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the network to retrieve. Conflicts with `name`, `machine_manager_id`, and `pool_id`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the network to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The ID of the machine manager to filter networks by. Required when searching by `name`.",
			},
			"pool_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The ID of the pool to filter networks by. Required when searching by `name`.",
			},

			// Out
			"machine_manager_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the machine manager this network belongs to.",
			},
			"machine_manager_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the machine manager this network belongs to.",
			},
			"internal_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internal identifier of the network in the Open IaaS system.",
			},
			"maximum_transmission_unit": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The Maximum Transmission Unit (MTU) size in bytes for this network.",
			},
			"network_adapters": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of network adapter IDs connected to this network.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"network_block_device": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this network supports network block devices.",
			},
			"insecure_network_block_device": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this network allows insecure network block devices.",
			},
		},
	}
}

// computeOpenIaaSNetworkRead lit un réseau OpenIaaS et le mappe dans le state Terraform
func computeOpenIaaSNetworkRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var network *client.OpenIaaSNetwork
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		networks, err := c.Compute().OpenIaaS().Network().List(ctx, &client.OpenIaaSNetworkFilter{
			MachineManagerID: d.Get("machine_manager_id").(string),
			PoolID:           d.Get("pool_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find network named %q: %s", name, err))
		}
		for _, n := range networks {
			if n.Name == name {
				network = n
				break
			}
		}
		if network == nil {
			return diag.FromErr(fmt.Errorf("failed to find network named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			network, err = c.Compute().OpenIaaS().Network().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if network == nil {
				return diag.FromErr(fmt.Errorf("failed to find network with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(network.ID)

	// Mapper les données en utilisant la fonction helper
	networkData := helpers.FlattenOpenIaaSNetwork(network)

	// Définir les données dans le state
	for k, v := range networkData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
