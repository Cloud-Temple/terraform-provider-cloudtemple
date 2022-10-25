package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualSwitchs() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceVirtualSwitchsRead,

		Schema: map[string]*schema.Schema{
			"virtual_switchs": {
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
						"folder_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceVirtualSwitchsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	switches, err := client.Compute().VirtualSwitch().List(ctx, "", "", "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(switches))
	for i, sw := range switches {
		res[i] = map[string]interface{}{
			"id":                 sw.ID,
			"name":               sw.Name,
			"moref":              sw.Moref,
			"folder_id":          sw.FolderID,
			"machine_manager_id": sw.MachineManagerID,
		}
	}

	sw := newStateWriter(d, "virtual-switches")
	sw.set("virtual_switchs", res)

	return sw.diags
}
