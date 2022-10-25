package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePersonalAccessToken() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, []string, error) {
			id := d.Get("client_id").(string)
			token, err := client.IAM().PAT().Read(ctx, id)
			if err == nil && token == nil {
				return nil, nil, fmt.Errorf("failed to find personal access token with id %q", id)
			}
			if token != nil {
				d.SetId(token.ID)
			}
			return token, []string{"secret"}, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"client_id": {
				Description: "",
				Type:        schema.TypeString,
				Required:    true,
			},

			// Out
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
	}
}
