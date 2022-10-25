package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceNetworkRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
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
	}
}

func dataSourceNetworkRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	network, err := client.Compute().Network().Read(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, network.ID)
	sw.set("id", network.ID)
	sw.set("name", network.Name)
	sw.set("moref", network.Moref)
	sw.set("machine_manager_id", network.MachineManagerId)
	sw.set("virtual_machines_number", network.VirtualMachinesNumber)
	sw.set("host_number", network.HostNumber)
	sw.set("host_names", network.HostNames)

	return sw.diags
}
