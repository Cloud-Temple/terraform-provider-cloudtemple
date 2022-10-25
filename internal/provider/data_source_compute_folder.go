package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFolder() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceFolderRead,

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
		},
	}
}

func dataSourceFolderRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	folder, err := client.Compute().Folder().Read(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, folder.ID)
	sw.set("name", folder.Name)
	sw.set("machine_manager_id", folder.MachineManagerId)

	return sw.diags
}
