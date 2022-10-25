package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHostClusters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error) {
			hcs, err := client.Compute().HostCluster().List(ctx, "", "", "")
			return map[string]interface{}{
				"id":            "host_clusters",
				"host_clusters": hcs,
			}, err
		}),

		Schema: map[string]*schema.Schema{
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
