package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualDatacenter() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceVirtualDatacenterRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
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
	}
}

func dataSourceVirtualDatacenterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	datacenter, err := client.Compute().VirtualDatacenter().Read(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, datacenter.ID)
	sw.set("id", datacenter.ID)
	sw.set("name", datacenter.Name)
	sw.set("machine_manager_id", datacenter.MachineManagerID)
	sw.set("tenant_id", datacenter.TenantID)

	return sw.diags
}
