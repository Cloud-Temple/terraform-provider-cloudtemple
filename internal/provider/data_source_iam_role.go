package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceRole() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				role, err := client.IAM().Role().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if role == nil {
					return nil, fmt.Errorf("failed to find role with id %q", id)
				}
				return role, nil
			}

			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				roles, err := client.IAM().Role().List(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to list roles: %s", err)
				}
				for _, role := range roles {
					if role.Name == name {
						return role, nil
					}
				}
				return nil, fmt.Errorf("failed to find role with name %q", name)
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
		},
	}
}
