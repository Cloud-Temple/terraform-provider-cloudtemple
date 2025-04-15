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

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific network in a virtual infrastructure.",

		ReadContext: computeNetworkRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the network to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				Description:   "The name of the network to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				Description:   "Filter networks by the ID of the machine manager they belong to. Only used when searching by `name`.",
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				Description:   "Filter networks by the ID of the datacenter they belong to. Only used when searching by `name`.",
			},
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				Description:   "Filter networks by the ID of a virtual machine connected to them. Only used when searching by `name`.",
			},
			"type": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.StringInSlice([]string{"Network", "DistributedVirtualPortgroup"}, false),
				ConflictsWith: []string{"id"},
				Description:   "Filter networks by type (Network or DistributedVirtualPortgroup). Only used when searching by `name`.",
			},
			"virtual_switch_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				Description:   "Filter networks by the ID of the virtual switch they are connected to. Only used when searching by `name`.",
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				Description:   "Filter networks by the ID of a host they are connected to. Only used when searching by `name`.",
			},
			"folder_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				Description:   "Filter networks by the ID of the folder they belong to. Only used when searching by `name`.",
			},
			"host_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				Description:   "Filter networks by the ID of the host cluster they are connected to. Only used when searching by `name`.",
			},

			// Out
			"moref": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The managed object reference ID of the network in the hypervisor.",
			},
			"virtual_machines_number": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of virtual machines connected to this network.",
			},
			"host_number": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of hosts connected to this network.",
			},
			"host_names": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of host names connected to this network.",

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

// computeNetworkRead lit un réseau et le mappe dans le state Terraform
func computeNetworkRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var network *client.Network
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		networks, err := c.Compute().Network().List(ctx, &client.NetworkFilter{
			Name:             name,
			MachineManagerId: d.Get("machine_manager_id").(string),
			DatacenterId:     d.Get("datacenter_id").(string),
			VirtualMachineId: d.Get("virtual_machine_id").(string),
			Type:             d.Get("type").(string),
			VirtualSwitchId:  d.Get("virtual_switch_id").(string),
			HostId:           d.Get("host_id").(string),
			FolderId:         d.Get("folder_id").(string),
			HostClusterId:    d.Get("host_cluster_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual network named %q: %s", name, err))
		}
		for _, n := range networks {
			if n.Name == name {
				network = n
				break
			}
		}
		if network == nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual network named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		network, err = c.Compute().Network().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if network == nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual network with id %q", id))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(network.ID)

	// Mapper les données en utilisant la fonction helper
	networkData := helpers.FlattenNetwork(network)

	// Définir les données dans le state
	for k, v := range networkData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
