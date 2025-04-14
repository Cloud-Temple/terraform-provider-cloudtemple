package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceContentLibraryItem() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: computeContentLibraryItemRead,

		Schema: map[string]*schema.Schema{
			// In
			"content_library_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},

			// Out
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
	}
}

// computeContentLibraryItemRead lit un élément d'une bibliothèque de contenu et le mappe dans le state Terraform
func computeContentLibraryItemRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var item *client.ContentLibraryItem
	var err error

	contentLibraryId := d.Get("content_library_id").(string)

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		items, err := c.Compute().ContentLibrary().ListItems(ctx, &client.ContentLibraryItemFilter{
			Name:             name,
			ContentLibraryId: contentLibraryId,
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find content library item named %q: %s", name, err))
		}
		for _, i := range items {
			if i.Name == name {
				item = i
				break
			}
		}
		if item == nil {
			return diag.FromErr(fmt.Errorf("failed to find content library item named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		item, err = c.Compute().ContentLibrary().ReadItem(ctx, contentLibraryId, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if item == nil {
			return diag.FromErr(fmt.Errorf("failed to find content library item with id %q", id))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(item.ID)

	// Mapper les données en utilisant la fonction helper
	itemData := helpers.FlattenContentLibraryItem(item)

	// Définir les données dans le state
	for k, v := range itemData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
