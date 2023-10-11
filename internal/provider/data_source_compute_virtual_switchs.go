package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualSwitchs() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			switches, err := c.Compute().VirtualSwitch().List(ctx, &client.VirtualSwitchFilter{
				Name:             d.Get("name").(string),
				MachineManagerId: d.Get("machine_manager_id").(string),
				DatacenterId:     d.Get("datacenter_id").(string),
				HostClusterId:    d.Get("host_cluster_id").(string),
			})
			return map[string]interface{}{
				"id":              "virtual_switchs",
				"virtual_switchs": switches,
			}, err
		}),

		Schema: map[string]*schema.Schema{
			// Out
			"virtual_switchs": {
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
						"moref": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"folder_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}
