package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePersonalAccessTokens() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourcePersonalAccessTokensRead,

		Schema: map[string]*schema.Schema{
			// In
			"user_id": {
				Description: "",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"tenant_id": {
				Description: "",
				Type:        schema.TypeString,
				Optional:    true,
			},

			// Out
			"tokens": {
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
						"roles": {
							Description: "",
							Type:        schema.TypeList,
							Computed:    true,

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func dataSourcePersonalAccessTokensRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	userID, diags := getUserID(ctx, client, d)
	if diags != nil {
		return diags
	}
	tenantID, diags := getTenantID(ctx, client, d)
	if diags != nil {
		return diags
	}

	tokens, err := client.IAM().PAT().List(ctx, userID, tenantID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("tokens")

	sw := newStateWriter(d)

	mTokens := []interface{}{}
	for _, t := range tokens {
		roles := []interface{}{}
		for _, r := range t.Roles {
			roles = append(roles, r)
		}

		mTokens = append(mTokens, map[string]interface{}{
			"id":    t.ID,
			"name":  t.Name,
			"roles": roles,
		})
	}

	sw.set("tokens", mTokens)

	return nil
}
