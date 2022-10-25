package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDatastoreClusters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceDatastoreClustersRead,

		Schema: map[string]*schema.Schema{
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

func dataSourceDatastoreClustersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	datastoreClusters, err := client.Compute().DatastoreCluster().List(ctx, "", "", "", "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(datastoreClusters))
	for i, dc := range datastoreClusters {
		res[i] = map[string]interface{}{
			"id":                 dc.ID,
			"name":               dc.Name,
			"moref":              dc.Moref,
			"machine_manager_id": dc.MachineManagerId,
			"datastores":         dc.Datastores,
			"metrics": []interface{}{
				map[string]interface{}{
					"free_capacity":                    dc.Metrics.FreeCapacity,
					"max_capacity":                     dc.Metrics.MaxCapacity,
					"enabled":                          dc.Metrics.Enabled,
					"default_vm_behavior":              dc.Metrics.DefaultVmBehavior,
					"load_balance_interval":            dc.Metrics.LoadBalanceInterval,
					"space_threshold_mode":             dc.Metrics.SpaceThresholdMode,
					"space_utilization_threshold":      dc.Metrics.SpaceUtilizationThreshold,
					"min_space_utilization_difference": dc.Metrics.MinSpaceUtilizationDifference,
					"reservable_percent_threshold":     dc.Metrics.ReservablePercentThreshold,
					"reservable_threshold_mode":        dc.Metrics.ReservableThresholdMode,
					"io_latency_threshold":             dc.Metrics.IoLatencyThreshold,
					"io_load_imbalance_threshold":      dc.Metrics.IoLoadImbalanceThreshold,
					"io_load_balance_enabled":          dc.Metrics.IoLoadBalanceEnabled,
				},
			},
		}
	}

	sw := newStateWriter(d, "datastore-clusters")
	sw.set("datastore_clusters", res)

	return sw.diags
}
