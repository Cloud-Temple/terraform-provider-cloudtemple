package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFolders() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceFoldersRead,

		Schema: map[string]*schema.Schema{
			"folders": {
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
					},
				},
			},
		},
	}
}

func dataSourceFoldersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	folders, err := client.Compute().Folder().List(ctx, "", "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(folders))
	for i, f := range folders {
		res[i] = map[string]interface{}{
			"id":                 f.ID,
			"name":               f.Name,
			"machine_manager_id": f.MachineManagerId,
		}
	}

	sw := newStateWriter(d, "folders")
	sw.set("folders", res)

	return sw.diags
}
