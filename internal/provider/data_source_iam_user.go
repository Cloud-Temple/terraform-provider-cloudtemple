package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error) {
			id := d.Get("id").(string)
			user, err := client.IAM().User().Read(ctx, id)
			if err == nil && user == nil {
				return nil, fmt.Errorf("failed to find user with id %q", id)
			}
			return user, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Description: "",
				Type:        schema.TypeString,
				Required:    true,
			},

			// Out
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
	}
}
