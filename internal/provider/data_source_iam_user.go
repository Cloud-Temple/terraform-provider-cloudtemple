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

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve information about a specific user.",

		ReadContext: dataSourceUserRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Description:   "The ID of the user.",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"internal_id", "name", "email"},
				ValidateFunc:  validation.IsUUID,
			},
			"internal_id": {
				Description:   "The internal ID of the user.",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"id", "name", "email"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Description:   "The name of the user.",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"id", "internal_id", "email"},
			},
			"email": {
				Description:   "The email of the user.",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"id", "internal_id", "name"},
			},

			// Out
			"type": {
				Description: "The type of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"source": {
				Description: "The source of the user.",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"source_id": {
				Description: "The source ID of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"email_verified": {
				Description: "Whether the user's email is verified.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
		},
	}
}

// dataSourceUserRead lit un utilisateur et le mappe dans le state Terraform
func dataSourceUserRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var user *client.User
	var err error

	// Recherche par ID
	id := d.Get("id").(string)
	if id != "" {
		user, err = c.IAM().User().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if user == nil {
			return diag.FromErr(fmt.Errorf("failed to find user with id %q", id))
		}
	} else {
		// Obtenir la liste des utilisateurs
		companyID, err := getCompanyID(ctx, c, d)
		if err != nil {
			return diag.FromErr(err)
		}
		users, err := c.IAM().User().List(ctx, &client.UserFilter{
			CompanyID: companyID,
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to list users: %s", err))
		}

		// Recherche par internal_id
		internalID := d.Get("internal_id").(string)
		if internalID != "" {
			for _, u := range users {
				if u.InternalID == internalID {
					user = u
					break
				}
			}
			if user == nil {
				return diag.FromErr(fmt.Errorf("failed to find user with internal_id %q", internalID))
			}
		} else {
			// Recherche par name
			name := d.Get("name").(string)
			if name != "" {
				for _, u := range users {
					if u.Name == name {
						user = u
						break
					}
				}
				if user == nil {
					return diag.FromErr(fmt.Errorf("failed to find user with name %q", name))
				}
			} else {
				// Recherche par email
				email := d.Get("email").(string)
				if email != "" {
					for _, u := range users {
						if u.Email == email {
							user = u
							break
						}
					}
					if user == nil {
						return diag.FromErr(fmt.Errorf("failed to find user with email %q", email))
					}
				} else {
					return diag.FromErr(fmt.Errorf("either id, internal_id, name or email must be specified"))
				}
			}
		}
	}

	// Définir l'ID de la datasource
	d.SetId(user.ID)

	// Mapper les données en utilisant la fonction helper
	userData := helpers.FlattenUser(user)

	// Définir les données dans le state
	for k, v := range userData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
