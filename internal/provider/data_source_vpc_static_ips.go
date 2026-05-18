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

func dataSourceVPCStaticIPs() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of static IPs for a specific private network.",

		ReadContext: dataSourceVPCStaticIPsRead,

		Schema: map[string]*schema.Schema{
			// In
			"private_network_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the private network to retrieve static IPs for.",
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
				Description: "List of static IPs matching the filter criteria.",

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
						"virtual_machine_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine associated with this static IP.",
						},
						"network_adapter_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the network adapter associated with this static IP.",
						},
						"source": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The source of the virtual machine (xoa, vmware, or custom).",
						},
						"resource_description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the resource (for custom sources).",
						},
						"floating_ip_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the floating IP bound to this static IP.",
						},
						"floating_ip_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The address of the floating IP bound to this static IP.",
						},
						"vpc_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the VPC.",
						},
						"private_network_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the private network.",
						},
					},
				},
			},
		},
	}
}

func dataSourceVPCStaticIPsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	privateNetworkID := d.Get("private_network_id").(string)
	virtualMachineID := d.Get("virtual_machine_id").(string)

	staticIPs, err := c.VPC().StaticIP().List(ctx, privateNetworkID, &client.StaticIPFilter{
		VirtualMachineID: virtualMachineID,
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to retrieve static IPs for private network %q: %s", privateNetworkID, err))
	}

	d.SetId("static_ips")

	// Map data using the helper function
	tfStaticIPs := make([]map[string]interface{}, len(staticIPs))
	for i, staticIP := range staticIPs {
		tfStaticIPs[i] = helpers.FlattenStaticIP(staticIP)
	}

	if err := d.Set("static_ips", tfStaticIPs); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
