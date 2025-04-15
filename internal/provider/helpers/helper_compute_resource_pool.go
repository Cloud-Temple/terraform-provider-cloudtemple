package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenResourcePool convertit un objet ResourcePool en une map compatible avec le schéma Terraform
func FlattenResourcePool(pool *client.ResourcePool) map[string]interface{} {
	// Mapper le parent
	parent := []map[string]interface{}{
		{
			"id":   pool.Parent.ID,
			"type": pool.Parent.Type,
		},
	}

	// Mapper les métriques CPU
	cpuMetrics := []map[string]interface{}{
		{
			"max_usage":        pool.Metrics.CPU.MaxUsage,
			"reservation_used": pool.Metrics.CPU.ReservationUsed,
		},
	}

	// Mapper les métriques mémoire
	memoryMetrics := []map[string]interface{}{
		{
			"max_usage":        pool.Metrics.Memory.MaxUsage,
			"reservation_used": pool.Metrics.Memory.ReservationUsed,
			"ballooned_memory": pool.Metrics.Memory.BalloonedMemory,
		},
	}

	// Mapper les métriques
	metrics := []map[string]interface{}{
		{
			"cpu":    cpuMetrics,
			"memory": memoryMetrics,
		},
	}

	return map[string]interface{}{
		"id":      pool.ID,
		"name":    pool.Name,
		"moref":   pool.Moref,
		"parent":  parent,
		"metrics": metrics,
		// "machine_manager_id": pool.MachineManager.ID,
		"machine_manager_id": pool.MachineManagerID, // DEPRECATED
	}
}
