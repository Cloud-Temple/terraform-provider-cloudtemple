package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualSwitch() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceVirtualSwitchRead,

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
			"folder_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceVirtualSwitchRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	swtch, err := client.Compute().VirtualSwitch().Read(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, swtch.ID)
	sw.set("name", swtch.Name)
	sw.set("moref", swtch.Moref)
	sw.set("folder_id", swtch.FolderID)
	sw.set("machine_manager_id", swtch.MachineManagerID)

	return sw.diags
}
