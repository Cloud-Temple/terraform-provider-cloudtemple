package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePersonalAccessTokens() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, []string, error) {
			userID, err := getUserID(ctx, client, d)
			if err != nil {
				return nil, nil, err
			}
			tenantID, err := getTenantID(ctx, client, d)
			if err != nil {
				return nil, nil, err
			}

			tokens, err := client.IAM().PAT().List(ctx, userID, tenantID)
			return map[string]interface{}{
				"id":     "tokens",
				"tokens": tokens,
			}, []string{"tokens.#.secret"}, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"user_id": {
				Description:  "",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"tenant_id": {
				Description:  "",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
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
						"expiration_date": {
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
