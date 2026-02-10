package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceContentLibraries() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of content libraries.",

		ReadContext: computeContentLibrariesRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter content libraries by name.",
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter content libraries by machine manager ID.",
			},

			// Out
			"content_libraries": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of content libraries matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the content library.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the content library.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this content library belongs to.",
						},
						"datastore": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the datastore associated with this content library.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the datastore.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the datastore.",
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
