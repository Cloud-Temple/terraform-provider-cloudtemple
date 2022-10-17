package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDatastoreClusters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			datastoreClusters, err := client.Compute().DatastoreCluster().List(ctx, "", "", "", "")
			return map[string]interface{}{
				"id":                 "datastore_clusters",
				"datastore_clusters": datastoreClusters,
			}, err
		}),

		Schema: map[string]*schema.Schema{
			// Out
			"datastore_clusters": {
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
						"datastores": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"metrics": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"free_capacity": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"max_capacity": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"default_vm_behavior": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"load_balance_interval": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"space_threshold_mode": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"space_utilization_threshold": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"min_space_utilization_difference": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"reservable_percent_threshold": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"reservable_threshold_mode": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"io_latency_threshold": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"io_load_imbalance_threshold": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"io_load_balance_enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
