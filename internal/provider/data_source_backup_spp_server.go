package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceBackupSPPServer() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			return getBy(
				ctx,
				d,
				"SPP server",
				func(id string) (any, error) {
					return client.Backup().SPPServer().Read(ctx, id)
				},
				func(d *schema.ResourceData) (any, error) {
					tenantId, err := getTenantID(ctx, client, d)
					if err != nil {
						return nil, err
					}
					return client.Backup().SPPServer().List(ctx, tenantId)
				},
				[]string{"name"},
			)
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"tenant_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},

			// Out
			"address": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
