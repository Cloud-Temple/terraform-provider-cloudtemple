package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNetworks() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			networks, err := c.Compute().Network().List(ctx, &client.NetworkFilter{
				Name:             d.Get("name").(string),
				MachineManagerId: d.Get("machine_manager_id").(string),
				DatacenterId:     d.Get("datacenter_id").(string),
				VirtualMachineId: d.Get("virtual_machine_id").(string),
				Type:             d.Get("type").(string),
				VirtualSwitchId:  d.Get("virtual_switch_id").(string),
				HostId:           d.Get("host_id").(string),
				FolderId:         d.Get("folder_id").(string),
				HostClusterId:    d.Get("host_cluster_id").(string),
			})
			return map[string]interface{}{
				"id":       "networks",
				"networks": networks,
			}, err
		}),

		Schema: map[string]*schema.Schema{
			// Out
			"networks": {
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
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_machines_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"host_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"host_names": {
							Type:     schema.TypeList,
							Computed: true,

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
