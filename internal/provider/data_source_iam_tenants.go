package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceTenants() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error) {
			tenants, err := client.IAM().Tenant().List(ctx)
			return map[string]interface{}{
				"id":      "tenants",
				"tenants": tenants,
			}, err
		}),

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
