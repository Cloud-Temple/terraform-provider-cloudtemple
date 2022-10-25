package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUsers() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error) {
			companyID, err := getCompanyID(ctx, client, d)
			if err != nil {
				return nil, err
			}

			users, err := client.IAM().User().List(ctx, companyID)
			return map[string]interface{}{
				"id":    "users",
				"users": users,
			}, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"company_id": {
				Description: "",
				Type:        schema.TypeString,
				Optional:    true,
			},

			// Out
			"users": {
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
						"internal_id": {
							Description: "",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "",
							Type:        schema.TypeString,
							Computed:    true,
						},
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
						"email": {
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
