package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceTenants() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceTenantsRead,

		Schema: map[string]*schema.Schema{
			// Out
			"tenants": {
				Description: "",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"snc": {
							Description: "",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"company_id": {
							Description: "",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceTenantsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	tenants, err := client.IAM().Tenant().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	lTenants := []interface{}{}
	for _, t := range tenants {
		lTenants = append(lTenants, map[string]interface{}{
			"id":         t.ID,
			"name":       t.Name,
			"snc":        t.SNC,
			"company_id": t.CompanyID,
		})
	}

	sw := newStateWriter(d, "tenants")
	sw.set("tenants", lTenants)

	return sw.diags
}
