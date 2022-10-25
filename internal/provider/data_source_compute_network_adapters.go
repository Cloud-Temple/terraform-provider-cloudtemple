package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNetworkAdapters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceNetworkAdaptersRead,

		Schema: map[string]*schema.Schema{
			"virtual_machine_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"network_adapters": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_machine_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"mac_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"mac_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"connected": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"auto_connect": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceNetworkAdaptersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	networkAdapters, err := client.Compute().NetworkAdapter().List(ctx, d.Get("virtual_machine_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(networkAdapters))
	for i, networkAdapter := range networkAdapters {
		res[i] = map[string]interface{}{
			"id":                 networkAdapter.ID,
			"virtual_machine_id": networkAdapter.VirtualMachineId,
			"name":               networkAdapter.Name,
			"type":               networkAdapter.Type,
			"mac_type":           networkAdapter.MacType,
			"mac_address":        networkAdapter.MacAddress,
			"connected":          networkAdapter.Connected,
			"auto_connect":       networkAdapter.AutoConnect,
		}
	}

	sw := newStateWriter(d, "network-adapters")
	sw.set("network_adapters", res)

	return sw.diags
}
