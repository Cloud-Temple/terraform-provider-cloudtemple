package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceUsers() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all users in a company.",

		ReadContext: dataSourceUsersRead,

		Schema: map[string]*schema.Schema{
			// In
			"company_id": {
				Description:  "The ID of the company to retrieve users for.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},

			// Out
			"users": {
				Description: "The list of users.",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The ID of the user.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"internal_id": {
							Description: "The internal ID of the user.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "The name of the user.",
							Type:        schema.TypeString,
							Computed:    true,
						},
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
						"email": {
							Description: "The email of the user.",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// dataSourceUsersRead lit les utilisateurs et les mappe dans le state Terraform
func dataSourceUsersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Obtenir l'ID de la company
	companyID, err := getCompanyID(ctx, c, d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Récupérer les utilisateurs
	users, err := c.IAM().User().List(ctx, &client.UserFilter{
		CompanyID: companyID,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("users")

	// Mapper les données en utilisant la fonction helper
	tfUsers := make([]map[string]interface{}, len(users))
	for i, user := range users {
		tfUsers[i] = helpers.FlattenUser(user)
	}

	// Définir les données dans le state
	if err := d.Set("users", tfUsers); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
