package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePersonalAccessToken() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, []string, error) {
			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				token, err := client.IAM().PAT().Read(ctx, id)
				if err != nil {
					return nil, []string{"secret"}, err
				}
				if token == nil {
					return nil, []string{"secret"}, fmt.Errorf("failed to find personal access token with id %q", id)
				}
				return token, []string{"secret"}, nil
			}

			// Obtenir les IDs utilisateur et tenant
			userId, err := getUserID(ctx, client, d)
			if err != nil {
				return nil, []string{"secret"}, err
			}
			tenantId, err := getTenantID(ctx, client, d)
			if err != nil {
				return nil, []string{"secret"}, err
			}

			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				tokens, err := client.IAM().PAT().List(ctx, userId, tenantId)
				if err != nil {
					return nil, []string{"secret"}, fmt.Errorf("failed to list personal access tokens: %s", err)
				}
				for _, token := range tokens {
					if token.Name == name {
						return token, []string{"secret"}, nil
					}
				}
				return nil, []string{"secret"}, fmt.Errorf("failed to find personal access token with name %q", name)
			}

			return nil, []string{"secret"}, fmt.Errorf("either id or name must be specified")
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"user_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
			},
			"tenant_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
			},

			// Out
			"roles": {
				Description: "",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"expiration_date": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}
