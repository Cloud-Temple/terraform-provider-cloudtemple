package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVirtualDatacenter() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			return getBy(
				ctx,
				d,
				"virtual datacenter",
				func(id string) (any, error) {
					return c.Compute().VirtualDatacenter().Read(ctx, id)
				},
				func(d *schema.ResourceData) (any, error) {
					return c.Compute().VirtualDatacenter().List(ctx, &client.VirtualDatacenterFilter{
						Name:             d.Get("name").(string),
						MachineManagerId: d.Get("machine_manager_id").(string),
					})
				},
				[]string{"name"},
			)
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},

			// Out
			"tenant_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
