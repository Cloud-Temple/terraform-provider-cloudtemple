package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualDatacenters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, []string, error) {
			datacenters, err := client.Compute().VirtualDatacenter().List(ctx, "", "")
			return map[string]interface{}{
				"id":                  "virtual_datacenters",
				"virtual_datacenters": datacenters,
			}, nil, err
		}),

		Schema: map[string]*schema.Schema{
			// Out
			"virtual_datacenters": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"tenant_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}
