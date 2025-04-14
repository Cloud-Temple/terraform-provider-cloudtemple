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
		Description: "",

		ReadContext: computeNetworksRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"datacenter_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"Network", "DistributedVirtualPortgroup"}, false),
			},
			"virtual_switch_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"folder_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},

			// Out
			"networks": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"moref": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_machines_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"host_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"host_names": {
							Type:     schema.TypeList,
							Computed: true,

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
