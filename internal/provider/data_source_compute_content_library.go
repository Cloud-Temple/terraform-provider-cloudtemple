package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceContentLibrary() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				contentLibraries, err := c.Compute().ContentLibrary().List(ctx, &client.ContentLibraryFilter{
					Name:             name,
					MachineManagerId: d.Get("machine_manager_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find content library named %q: %s", name, err)
				}
				for _, cl := range contentLibraries {
					if cl.Name == name {
						return cl, nil
					}
				}
				return nil, fmt.Errorf("failed to find host cluster named %q", name)
			}

			id := d.Get("id").(string)
			contentLibrary, err := c.Compute().ContentLibrary().Read(ctx, id)
			if err == nil && contentLibrary == nil {
				return nil, fmt.Errorf("failed to find host cluster with id %q", id)
			}
			return contentLibrary, err
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
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
			},

			// Out
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
	}
}
