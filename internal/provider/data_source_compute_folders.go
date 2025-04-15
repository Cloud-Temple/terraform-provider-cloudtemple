package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFolders() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of all folders.",

		ReadContext: computeFoldersRead,

		Schema: map[string]*schema.Schema{
			// Out
			"folders": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of all folders.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the folder.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the folder.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this folder belongs to.",
						},
					},
				},
			},
		},
	}
}

// computeFoldersRead lit les dossiers et les mappe dans le state Terraform
func computeFoldersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les dossiers
	folders, err := c.Compute().Folder().List(ctx, &client.FolderFilter{})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("folders")

	// Mapper manuellement les données en utilisant la fonction helper
	tfFolders := make([]map[string]interface{}, len(folders))
	for i, folder := range folders {
		tfFolders[i] = helpers.FlattenFolder(folder)
	}

	// Définir les données dans le state
	if err := d.Set("folders", tfFolders); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
