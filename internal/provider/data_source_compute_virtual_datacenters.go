package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualDatacenters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceVirtualDatacentersRead,

		Schema: map[string]*schema.Schema{
			"virtual_datacenters": {
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
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"tenant_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceVirtualDatacentersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	datacenters, err := client.Compute().VirtualDatacenter().List(ctx, "", "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(datacenters))
	for i, datacenter := range datacenters {
		res[i] = map[string]interface{}{
			"id":                 datacenter.ID,
			"name":               datacenter.Name,
			"machine_manager_id": datacenter.MachineManagerID,
			"tenant_id":          datacenter.TenantID,
		}
	}

	sw := newStateWriter(d, "virtual-datacenters")
	sw.set("virtual_datacenters", res)

	return sw.diags
}
