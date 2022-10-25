package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceNetworkAdapterRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
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
	}
}

func dataSourceNetworkAdapterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	networkAdapter, err := client.Compute().NetworkAdapter().Read(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, networkAdapter.ID)
	sw.set("virtual_machine_id", networkAdapter.VirtualMachineId)
	sw.set("name", networkAdapter.Name)
	sw.set("type", networkAdapter.Type)
	sw.set("mac_type", networkAdapter.MacType)
	sw.set("mac_address", networkAdapter.MacAddress)
	sw.set("connected", networkAdapter.Connected)
	sw.set("auto_connect", networkAdapter.AutoConnect)

	return sw.diags
}
