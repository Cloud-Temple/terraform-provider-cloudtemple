package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUsers() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceUsersRead,

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

func dataSourceUsersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	companyID, diags := getCompanyID(ctx, client, d)
	if diags != nil {
		return diags
	}

	users, err := client.IAM().User().List(ctx, companyID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("users")

	lUsers := []interface{}{}
	for _, u := range users {
		lUsers = append(lUsers, map[string]interface{}{
			"id":             u.ID,
			"internal_id":    u.InternalID,
			"name":           u.Name,
			"type":           u.Type,
			"source":         u.Source,
			"source_id":      u.SourceID,
			"email_verified": u.EmailVerified,
			"email":          u.Email,
		})
	}

	sw := newStateWriter(d)
	sw.set("users", lUsers)

	return sw.diags
}
