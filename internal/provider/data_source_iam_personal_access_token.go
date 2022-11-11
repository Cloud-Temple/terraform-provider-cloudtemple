package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePersonalAccessToken() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, []string, error) {
			token, err := getBy(
				ctx,
				d,
				"personal access token",
				func(id string) (any, error) {
					return client.IAM().PAT().Read(ctx, id)
				},
				func(d *schema.ResourceData) (any, error) {
					userId, err := getUserID(ctx, client, d)
					if err != nil {
						return nil, err
					}
					tenantId, err := getTenantID(ctx, client, d)
					if err != nil {
						return nil, err
					}
					return client.IAM().PAT().List(ctx, userId, tenantId)
				},
				[]string{"name"},
			)
			return token, []string{"secret"}, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
			},
			"name": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"user_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
			},
			"tenant_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
			},

			// Out
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
	}
}
