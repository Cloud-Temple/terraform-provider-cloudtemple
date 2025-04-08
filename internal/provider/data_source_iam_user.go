package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				user, err := client.IAM().User().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if user == nil {
					return nil, fmt.Errorf("failed to find user with id %q", id)
				}
				return user, nil
			}

			// Obtenir la liste des utilisateurs
			companyId, err := getCompanyID(ctx, client, d)
			if err != nil {
				return nil, err
			}
			users, err := client.IAM().User().List(ctx, companyId)
			if err != nil {
				return nil, fmt.Errorf("failed to list users: %s", err)
			}

			// Recherche par internal_id
			internalId := d.Get("internal_id").(string)
			if internalId != "" {
				for _, user := range users {
					if user.InternalID == internalId {
						return user, nil
					}
				}
				return nil, fmt.Errorf("failed to find user with internal_id %q", internalId)
			}

			// Recherche par name
			name := d.Get("name").(string)
			if name != "" {
				for _, user := range users {
					if user.Name == name {
						return user, nil
					}
				}
				return nil, fmt.Errorf("failed to find user with name %q", name)
			}

			// Recherche par email
			email := d.Get("email").(string)
			if email != "" {
				for _, user := range users {
					if user.Email == email {
						return user, nil
					}
				}
				return nil, fmt.Errorf("failed to find user with email %q", email)
			}

			return nil, fmt.Errorf("either id, internal_id, name or email must be specified")
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"internal_id", "name", "email"},
				ValidateFunc:  validation.IsUUID,
			},
			"internal_id": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"id", "name", "email"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"id", "internal_id", "email"},
			},
			"email": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"id", "internal_id", "name"},
			},

			// Out
			"type": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"source": {
				Description: "",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"source_id": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"email_verified": {
				Description: "",
				Type:        schema.TypeBool,
				Computed:    true,
			},
		},
	}
}
