package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupSites() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: backupSitesRead,

		Schema: map[string]*schema.Schema{
			// Out
			"sites": {
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
					},
				},
			},
		},
	}
}

// backupSitesRead lit les sites de backup et les mappe dans le state Terraform
func backupSitesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les sites
	sites, err := c.Backup().Site().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("sites")

	// Mapper manuellement les données en utilisant la fonction helper
	tfSites := make([]map[string]interface{}, len(sites))
	for i, site := range sites {
		tfSites[i] = helpers.FlattenBackupSite(site)
	}

	// Définir les données dans le state
	if err := d.Set("sites", tfSites); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
