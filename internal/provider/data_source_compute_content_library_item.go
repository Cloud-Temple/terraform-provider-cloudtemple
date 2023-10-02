package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceContentLibraryItem() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				items, err := c.Compute().ContentLibrary().ListItems(ctx, &client.ContentLibraryItemFilter{
					Name:             name,
					ContentLibraryId: d.Get("content_library_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find content library item named %q: %s", name, err)
				}
				for _, item := range items {
					if item.Name == name {
						return item, nil
					}
				}
				return nil, fmt.Errorf("failed to find content library item named %q", name)
			}

			id := d.Get("id").(string)
			item, err := c.Compute().ContentLibrary().ReadItem(
				ctx,
				d.Get("content_library_id").(string),
				id,
			)
			if err == nil && item == nil {
				return nil, fmt.Errorf("failed to find content library item with id %q", id)
			}
			return item, err
		}),

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
