package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			return getBy(
				ctx,
				d,
				"user",
				func(id string) (any, error) {
					return client.IAM().User().Read(ctx, id)
				},
				func(d *schema.ResourceData) (any, error) {
					companyId, err := getCompanyID(ctx, client, d)
					if err != nil {
						return nil, err
					}
					return client.IAM().User().List(ctx, companyId)
				},
				[]string{"internal_id", "name", "email"},
			)
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"internal_id", "name", "email"},
				ValidateFunc:  validation.IsUUID,
			},
			"internal_id": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"id", "name", "email"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"id", "internal_id", "email"},
			},
			"email": {
				Description:   "",
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "internal_id", "name", "email"},
				ConflictsWith: []string{"id", "internal_id", "name"},
			},

			// Out
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
		},
	}
}
