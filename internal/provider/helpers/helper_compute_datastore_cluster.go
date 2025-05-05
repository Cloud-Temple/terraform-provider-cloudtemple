package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenDatastoreCluster convertit un objet DatastoreCluster en une map compatible avec le schéma Terraform
func FlattenDatastoreCluster(datastoreCluster *client.DatastoreCluster) map[string]interface{} {
	// Mapper les métriques
	metrics := []map[string]interface{}{
		{
			"free_capacity":                    datastoreCluster.Metrics.FreeCapacity,
			"max_capacity":                     datastoreCluster.Metrics.MaxCapacity,
			"enabled":                          datastoreCluster.Metrics.Enabled,
			"default_vm_behavior":              datastoreCluster.Metrics.DefaultVmBehavior,
			"load_balance_interval":            datastoreCluster.Metrics.LoadBalanceInterval,
			"space_threshold_mode":             datastoreCluster.Metrics.SpaceThresholdMode,
			"space_utilization_threshold":      datastoreCluster.Metrics.SpaceUtilizationThreshold,
			"min_space_utilization_difference": datastoreCluster.Metrics.MinSpaceUtilizationDifference,
			"reservable_percent_threshold":     datastoreCluster.Metrics.ReservablePercentThreshold,
			"reservable_threshold_mode":        datastoreCluster.Metrics.ReservableThresholdMode,
			"io_latency_threshold":             datastoreCluster.Metrics.IoLatencyThreshold,
			"io_load_imbalance_threshold":      datastoreCluster.Metrics.IoLoadImbalanceThreshold,
			"io_load_balance_enabled":          datastoreCluster.Metrics.IoLoadBalanceEnabled,
		},
	}

	return map[string]interface{}{
		"id":                 datastoreCluster.ID,
		"name":               datastoreCluster.Name,
		"moref":              datastoreCluster.Moref,
		"machine_manager_id": datastoreCluster.MachineManager.ID,
		"datacenter_id":      datastoreCluster.Datacenter.ID,
		"datastores":         datastoreCluster.Datastores,
		"metrics":            metrics,
	}
}
