package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasNetworkAdapters() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all network adapters from an Open IaaS infrastructure for a specific virtual machine.",

		ReadContext: computeOpenIaaSNetworkAdaptersRead,

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
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the network adapter.",
						},
						"internal_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internal identifier of the network adapter in the Open IaaS system.",
						},
						"virtual_machine_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine this network adapter is attached to.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this network adapter belongs to.",
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
						"tx_checksumming": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether TX checksumming is enabled on the network adapter.",
						},
						"attached": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the network adapter is attached to a virtual machine.",
						},
						"ipv4_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The IPv4 address of the network adapter.",
						},
						"ipv6_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The IPv6 address of the network adapter.",
						},
						"vpc_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the VPC the network adapter is associated with, or an empty string when the adapter is not on a VPC network.",
						},
						"vpc_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the VPC the network adapter is associated with, or an empty string when the adapter is not on a VPC network.",
						},
						"private_network_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the VPC private network the network adapter is associated with, or an empty string when the adapter is not on a VPC network.",
						},
						"private_network_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the VPC private network the network adapter is associated with, or an empty string when the adapter is not on a VPC network.",
						},
						"static_ip_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The static IP address assigned to the network adapter on its VPC, or an empty string when the adapter is not on a VPC network.",
						},
					},
				},
			},
		},
	}
}

// computeOpenIaaSNetworkAdaptersRead lit les adaptateurs réseau OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSNetworkAdaptersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les adaptateurs réseau OpenIaaS
	virtualMachineId := d.Get("virtual_machine_id").(string)
	networkAdapters, err := c.Compute().OpenIaaS().NetworkAdapter().List(ctx, &client.OpenIaaSNetworkAdapterFilter{
		VirtualMachineID: virtualMachineId,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_network_adapters")

	// Mapper manuellement les données en utilisant la fonction helper
	tfNetworkAdapters := make([]map[string]interface{}, len(networkAdapters))
	for i, adapter := range networkAdapters {
		tfNetworkAdapters[i] = helpers.FlattenOpenIaaSNetworkAdapter(adapter)
		tfNetworkAdapters[i]["id"] = adapter.ID
	}

	// Définir les données dans le state
	if err := d.Set("network_adapters", tfNetworkAdapters); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
