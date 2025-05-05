package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceTenants() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all tenants in the platform.",

		ReadContext: dataSourceTenantsRead,

		Schema: map[string]*schema.Schema{
			// Out
			"tenants": {
				Description: "The list of tenants.",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The ID of the tenant.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "The name of the tenant.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"snc": {
							Description: "Whether the tenant is a SNC tenant.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"company_id": {
							Description: "The ID of the company that owns the tenant.",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// dataSourceTenantsRead lit les tenants et les mappe dans le state Terraform
func dataSourceTenantsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les tenants
	tenants, err := c.IAM().Tenant().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("tenants")

	// Mapper les données en utilisant la fonction helper
	tfTenants := make([]map[string]interface{}, len(tenants))
	for i, tenant := range tenants {
		tfTenants[i] = helpers.FlattenTenant(tenant)
	}

	// Définir les données dans le state
	if err := d.Set("tenants", tfTenants); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
