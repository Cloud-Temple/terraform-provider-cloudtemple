package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceNetworkAdapters() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all network adapters attached to a specific virtual machine.",

		ReadContext: computeNetworkAdaptersRead,

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the virtual machine to retrieve network adapters for.",
			},

			// Out
			"network_adapters": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of network adapters attached to the specified virtual machine.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the network adapter.",
						},
						"virtual_machine_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine this network adapter is attached to.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the network adapter.",
						},
						"network_id": {
							Type:        schema.TypeString,
							Required:    true,
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
				},
			},
		},
	}
}

// computeNetworkAdaptersRead lit les adaptateurs réseau et les mappe dans le state Terraform
func computeNetworkAdaptersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les adaptateurs réseau
	networkAdapters, err := c.Compute().NetworkAdapter().List(ctx, &client.NetworkAdapterFilter{
		VirtualMachineID: d.Get("virtual_machine_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("network_adapters")

	// Mapper manuellement les données en utilisant la fonction helper
	tfNetworkAdapters := make([]map[string]interface{}, len(networkAdapters))
	for i, adapter := range networkAdapters {
		tfNetworkAdapters[i] = helpers.FlattenNetworkAdapter(adapter)
		tfNetworkAdapters[i]["id"] = adapter.ID
	}

	// Définir les données dans le state
	if err := d.Set("network_adapters", tfNetworkAdapters); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
