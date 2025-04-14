package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRoles() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all available roles in the platform.",

		ReadContext: dataSourceRolesRead,

		Schema: map[string]*schema.Schema{
			// Out
			"roles": {
				Description: "The list of roles.",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The ID of the role.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "The name of the role.",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// dataSourceRolesRead lit les rôles et les mappe dans le state Terraform
func dataSourceRolesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les rôles
	roles, err := c.IAM().Role().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("roles")

	// Mapper les données en utilisant la fonction helper
	tfRoles := make([]map[string]interface{}, len(roles))
	for i, role := range roles {
		tfRoles[i] = helpers.FlattenRole(role)
	}

	// Définir les données dans le state
	if err := d.Set("roles", tfRoles); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
