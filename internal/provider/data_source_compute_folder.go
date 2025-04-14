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

func dataSourceFolder() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: computeFolderRead,

		Schema: map[string]*schema.Schema{
			// In
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
				RequiredWith:  []string{"datacenter_id"},
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name"},
			},
		},
	}
}

// computeFolderRead lit un dossier et le mappe dans le state Terraform
func computeFolderRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var folder *client.Folder
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		folders, err := c.Compute().Folder().List(ctx, &client.FolderFilter{
			Name:             name,
			DatacenterID:     d.Get("datacenter_id").(string),
			MachineManagerID: d.Get("machine_manager_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find folder named %q: %s", name, err))
		}
		for _, f := range folders {
			if f.Name == name {
				folder = f
				break
			}
		}
		if folder == nil {
			return diag.FromErr(fmt.Errorf("failed to find folder named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			folder, err = c.Compute().Folder().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if folder == nil {
				return diag.FromErr(fmt.Errorf("failed to find folder with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(folder.ID)

	// Mapper les données en utilisant la fonction helper
	folderData := helpers.FlattenFolder(folder)

	// Définir les données dans le state
	for k, v := range folderData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
