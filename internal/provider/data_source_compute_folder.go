package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceFolder() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				folders, err := client.Compute().Folder().List(ctx, "", "")
				if err != nil {
					return nil, fmt.Errorf("failed to find folder named %q: %s", name, err)
				}
				for _, folder := range folders {
					if folder.Name == name {
						return folder, nil
					}
				}
				return nil, fmt.Errorf("failed to find folder named %q", name)
			}

			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				folder, err := client.Compute().Folder().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if folder == nil {
					return nil, fmt.Errorf("failed to find folder with id %q", id)
				}
				return folder, nil
			}

			return nil, fmt.Errorf("either id or name must be specified")
		}),

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
			},

			// Out
			"machine_manager_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
