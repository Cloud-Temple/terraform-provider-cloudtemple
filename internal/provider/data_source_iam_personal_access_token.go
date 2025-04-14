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

func dataSourcePersonalAccessToken() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve information about a specific personal access token.",

		ReadContext: dataSourcePersonalAccessTokenRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Description:   "The ID of the personal access token.",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Description:   "The name of the personal access token.",
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
				Description: "The roles associated with the personal access token.",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"expiration_date": {
				Description: "The expiration date of the personal access token.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

// dataSourcePersonalAccessTokenRead lit un token et le mappe dans le state Terraform
func dataSourcePersonalAccessTokenRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var token *client.Token
	var err error

	// Recherche par ID
	id := d.Get("id").(string)
	if id != "" {
		token, err = c.IAM().PAT().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if token == nil {
			return diag.FromErr(fmt.Errorf("failed to find personal access token with id %q", id))
		}
	} else {
		// Obtenir les IDs utilisateur et tenant
		userId, err := getUserID(ctx, c, d)
		if err != nil {
			return diag.FromErr(err)
		}
		tenantId, err := getTenantID(ctx, c, d)
		if err != nil {
			return diag.FromErr(err)
		}

		// Recherche par nom
		name := d.Get("name").(string)
		if name != "" {
			tokens, err := c.IAM().PAT().List(ctx, userId, tenantId)
			if err != nil {
				return diag.FromErr(fmt.Errorf("failed to list personal access tokens: %s", err))
			}
			for _, t := range tokens {
				if t.Name == name {
					token = t
					break
				}
			}
			if token == nil {
				return diag.FromErr(fmt.Errorf("failed to find personal access token with name %q", name))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(token.ID)

	// Mapper les données en utilisant la fonction helper
	tokenData := helpers.FlattenToken(token)

	// Ne pas exposer le secret dans la datasource
	delete(tokenData, "secret")

	// Définir les données dans le state
	for k, v := range tokenData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
