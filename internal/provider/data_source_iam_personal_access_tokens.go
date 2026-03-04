package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePersonalAccessTokens() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all personal access tokens for a user in a tenant.",

		ReadContext: dataSourcePersonalAccessTokensRead,

		Schema: map[string]*schema.Schema{
			// Out
			"tokens": {
				Description: "The list of personal access tokens.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The ID of the personal access token.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "The name of the personal access token.",
							Type:        schema.TypeString,
							Computed:    true,
						},
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
						"user_id": {
							Description: "The ID of the user this token is related to.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"tenant_id": {
							Description: "The ID of the tenant this token is related to.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"tenant_name": {
							Description: "The name of the tenant this token is related to.",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// dataSourcePersonalAccessTokensRead lit les tokens et les mappe dans le state Terraform
func dataSourcePersonalAccessTokensRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les tokens
	tokens, err := c.IAM().PAT().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("tokens")

	// Mapper les données en utilisant la fonction helper
	tfTokens := make([]map[string]interface{}, len(tokens))
	for i, token := range tokens {
		tokenData := helpers.FlattenToken(token)
		tfTokens[i] = tokenData
	}

	// Définir les données dans le state
	if err := d.Set("tokens", tfTokens); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
