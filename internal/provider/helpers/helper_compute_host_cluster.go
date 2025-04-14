package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenHostCluster convertit un objet HostCluster en une map compatible avec le schéma Terraform
func FlattenHostCluster(hostCluster *client.HostCluster) map[string]interface{} {
	// Mapper les hosts
	hosts := make([]map[string]interface{}, len(hostCluster.Hosts))
	for i, host := range hostCluster.Hosts {
		hosts[i] = map[string]interface{}{
			"id":   host.ID,
			"type": host.Type,
		}
	}

	// Mapper les métriques
	metrics := []map[string]interface{}{
		{
			"total_cpu":     hostCluster.Metrics.TotalCpu,
			"total_memory":  hostCluster.Metrics.TotalMemory,
			"total_storage": hostCluster.Metrics.TotalStorage,
			"cpu_used":      hostCluster.Metrics.CpuUsed,
			"memory_used":   hostCluster.Metrics.MemoryUsed,
			"storage_used":  hostCluster.Metrics.StorageUsed,
		},
	}

	return map[string]interface{}{
		"id":                      hostCluster.ID,
		"name":                    hostCluster.Name,
		"moref":                   hostCluster.Moref,
		"hosts":                   hosts,
		"metrics":                 metrics,
		"virtual_machines_number": hostCluster.VirtualMachinesNumber,
		"machine_manager_id":      hostCluster.MachineManager.ID,
		"datacenter_id":           hostCluster.Datacenter.ID,
	}
}
