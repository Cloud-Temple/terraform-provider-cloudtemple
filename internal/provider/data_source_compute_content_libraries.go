package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceContentLibraries() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: computeContentLibrariesRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// Out
			"content_libraries": {
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
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"datastore": {
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
				},
			},
		},
	}
}

// computeContentLibrariesRead lit les bibliothèques de contenu et les mappe dans le state Terraform
func computeContentLibrariesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les bibliothèques de contenu
	contentLibraries, err := c.Compute().ContentLibrary().List(ctx, &client.ContentLibraryFilter{
		Name:             d.Get("name").(string),
		MachineManagerId: d.Get("machine_manager_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("content_libraries")

	// Mapper manuellement les données en utilisant la fonction helper
	tfContentLibraries := make([]map[string]interface{}, len(contentLibraries))
	for i, contentLibrary := range contentLibraries {
		tfContentLibraries[i] = helpers.FlattenContentLibrary(contentLibrary)
	}

	// Définir les données dans le state
	if err := d.Set("content_libraries", tfContentLibraries); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
