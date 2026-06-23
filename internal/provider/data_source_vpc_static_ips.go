package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVPCStaticIPs() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all static IPs of a VPC private network. Can be filtered by virtual machine ID.",

		ReadContext: dataSourceVPCStaticIPsRead,

		Schema: map[string]*schema.Schema{
			// In
			"private_network_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the private network whose static IPs to retrieve.",
			},
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter static IPs by virtual machine ID.",
			},

			// Out
			"static_ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of static IPs matching the filter.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the static IP.",
						},
						"ip_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The static IP address.",
						},
						"mac_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The MAC address of the network adapter.",
						},
						"source": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The source of the static IP (one of `xoa`, `vmware`, `custom`).",
						},
						"resource_description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the resource, for a custom source.",
						},
						"virtual_machine_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine associated with this static IP, if any.",
						},
						"network_adapter_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the network adapter associated with this static IP, if any.",
						},
						"floating_ip_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the floating IP bound to this static IP, if any.",
						},
						"floating_ip_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The address of the floating IP bound to this static IP, if any.",
						},
						"vpc_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the VPC this static IP belongs to.",
						},
						"private_network_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the private network this static IP belongs to.",
						},
					},
				},
			},
		},
	}
}

func dataSourceVPCStaticIPsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	privateNetworkID := d.Get("private_network_id").(string)
	virtualMachineID := d.Get("virtual_machine_id").(string)

	staticIPs, err := c.VPC().StaticIP().List(ctx, privateNetworkID, &client.StaticIPFilter{
		VirtualMachineID: virtualMachineID,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(privateNetworkID)

	tfStaticIPs := make([]map[string]interface{}, len(staticIPs))
	for i, staticIP := range staticIPs {
		tfStaticIPs[i] = helpers.FlattenStaticIP(staticIP)
	}

	if err := d.Set("static_ips", tfStaticIPs); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
