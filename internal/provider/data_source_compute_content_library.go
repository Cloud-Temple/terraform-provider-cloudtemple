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

func dataSourceContentLibrary() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific content library.",

		ReadContext: computeContentLibraryRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the content library to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				Description:   "The name of the content library to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				Description:   "The ID of the machine manager to filter content libraries by. Only used when searching by name.",
			},

			// Out
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the content library.",
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
	}
}

// computeContentLibraryRead lit une bibliothèque de contenu et la mappe dans le state Terraform
func computeContentLibraryRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var contentLibrary *client.ContentLibrary
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		contentLibraries, err := c.Compute().ContentLibrary().List(ctx, &client.ContentLibraryFilter{
			Name:             name,
			MachineManagerId: d.Get("machine_manager_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find content library named %q: %s", name, err))
		}
		for _, cl := range contentLibraries {
			if cl.Name == name {
				contentLibrary = cl
				break
			}
		}
		if contentLibrary == nil {
			return diag.FromErr(fmt.Errorf("failed to find content library named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		contentLibrary, err = c.Compute().ContentLibrary().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if contentLibrary == nil {
			return diag.FromErr(fmt.Errorf("failed to find content library with id %q", id))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(contentLibrary.ID)

	// Mapper les données en utilisant la fonction helper
	contentLibraryData := helpers.FlattenContentLibrary(contentLibrary)

	// Définir les données dans le state
	for k, v := range contentLibraryData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
