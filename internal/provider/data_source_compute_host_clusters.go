package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHostClusters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			hcs, err := c.Compute().HostCluster().List(ctx, &client.HostClusterFilter{
				Name:             d.Get("name").(string),
				MachineManagerId: d.Get("machine_manager_id").(string),
				DatacenterId:     d.Get("datacenter_id").(string),
				DatastoreId:      d.Get("datastore_id").(string),
			})
			return map[string]interface{}{
				"id":            "host_clusters",
				"host_clusters": hcs,
			}, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"datastore_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			// Out
			"host_clusters": {
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
						"hosts": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"type": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"metrics": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"total_cpu": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"total_memory": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"total_storage": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"cpu_used": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"memory_used": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"storage_used": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"virtual_machines_number": {
							Type:     schema.TypeInt,
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
