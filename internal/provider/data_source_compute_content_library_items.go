package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceContentLibraryItems() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: computeContentLibraryItemsRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"content_library_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},

			// Out
			"content_library_items": {
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
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"creation_time": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"stored": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"last_modified_time": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ovf_properties": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

// computeContentLibraryItemsRead lit les éléments d'une bibliothèque de contenu et les mappe dans le state Terraform
func computeContentLibraryItemsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les éléments de la bibliothèque de contenu
	contentLibraryItems, err := c.Compute().ContentLibrary().ListItems(ctx, &client.ContentLibraryItemFilter{
		Name:             d.Get("name").(string),
		ContentLibraryId: d.Get("content_library_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("content_library_items")

	// Mapper manuellement les données en utilisant la fonction helper
	tfContentLibraryItems := make([]map[string]interface{}, len(contentLibraryItems))
	for i, item := range contentLibraryItems {
		tfContentLibraryItems[i] = helpers.FlattenContentLibraryItem(item)
	}

	// Définir les données dans le state
	if err := d.Set("content_library_items", tfContentLibraryItems); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
