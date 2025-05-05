package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceNetworks() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all networks in a virtual infrastructure that match the given criteria.",

		ReadContext: computeNetworksRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter networks by name. Partial matches are supported.",
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter networks by the ID of the machine manager they belong to.",
			},
			"datacenter_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter networks by the ID of the datacenter they belong to.",
			},
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter networks by the ID of a virtual machine connected to them.",
			},
			"type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"Network", "DistributedVirtualPortgroup"}, false),
				Description:  "Filter networks by type (Network or DistributedVirtualPortgroup).",
			},
			"virtual_switch_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter networks by the ID of the virtual switch they are connected to.",
			},
			"host_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter networks by the ID of a host they are connected to.",
			},
			"folder_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter networks by the ID of the folder they belong to.",
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter networks by the ID of the host cluster they are connected to.",
			},

			// Out
			"networks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of networks matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the network.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the network.",
						},
						"moref": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The managed object reference ID of the network in the hypervisor.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this network belongs to.",
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
				},
			},
		},
	}
}

// computeNetworksRead lit les réseaux et les mappe dans le state Terraform
func computeNetworksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les réseaux
	networks, err := c.Compute().Network().List(ctx, &client.NetworkFilter{
		Name:             d.Get("name").(string),
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
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("networks")

	// Mapper manuellement les données en utilisant la fonction helper
	tfNetworks := make([]map[string]interface{}, len(networks))
	for i, network := range networks {
		tfNetworks[i] = helpers.FlattenNetwork(network)
	}

	// Définir les données dans le state
	if err := d.Set("networks", tfNetworks); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
