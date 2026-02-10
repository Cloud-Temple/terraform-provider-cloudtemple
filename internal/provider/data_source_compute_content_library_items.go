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
		Description: "Used to retrieve a list of items from a content library.",

		ReadContext: computeContentLibraryItemsRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter items by name.",
			},
			"content_library_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the content library to retrieve items from.",
			},

			// Out
			"content_library_items": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of content library items matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the content library item.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the content library item.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the content library item.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the content library item (e.g., OVF, ISO).",
						},
						"creation_time": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The timestamp when the content library item was created.",
						},
						"size": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The size of the content library item in bytes.",
						},
						"last_modified_time": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The timestamp when the content library item was last modified.",
						},
						"ovf_properties": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of OVF properties associated with the content library item.",

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
