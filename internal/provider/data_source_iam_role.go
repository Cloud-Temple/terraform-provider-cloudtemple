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

func dataSourceRole() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve information about a specific role.",

		ReadContext: dataSourceRoleRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Description:   "The ID of the role.",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Description:   "The name of the role.",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
		},
	}
}

// dataSourceRoleRead lit un rôle et le mappe dans le state Terraform
func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var role *client.Role
	var err error

	// Recherche par ID
	id := d.Get("id").(string)
	if id != "" {
		role, err = c.IAM().Role().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if role == nil {
			return diag.FromErr(fmt.Errorf("failed to find role with id %q", id))
		}
	} else {
		// Recherche par nom
		name := d.Get("name").(string)
		if name != "" {
			roles, err := c.IAM().Role().List(ctx)
			if err != nil {
				return diag.FromErr(fmt.Errorf("failed to list roles: %s", err))
			}
			for _, r := range roles {
				if r.Name == name {
					role = r
					break
				}
			}
			if role == nil {
				return diag.FromErr(fmt.Errorf("failed to find role with name %q", name))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(role.ID)

	// Mapper les données en utilisant la fonction helper
	roleData := helpers.FlattenRole(role)

	// Définir les données dans le state
	for k, v := range roleData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
