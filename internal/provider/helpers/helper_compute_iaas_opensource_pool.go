package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSPool convertit un objet OpenIaasPool en une map compatible avec le schéma Terraform
func FlattenOpenIaaSPool(pool *client.OpenIaasPool) map[string]interface{} {
	// Mapper le CPU
	cpu := []map[string]interface{}{
		{
			"cores":   pool.Cpu.Cores,
			"sockets": pool.Cpu.Sockets,
		},
	}

	// Mapper la memory
	memory := []map[string]interface{}{
		{
			"usage": pool.Memory.Usage,
			"size":  pool.Memory.Size,
		},
	}

	// Mapper le type
	poolType := []map[string]interface{}{
		{
			"key":         pool.Type.Key,
			"description": pool.Type.Description,
		},
	}

	return map[string]interface{}{
		"id":                        pool.ID,
		"name":                      pool.Name,
		"label":                     pool.Label,
		"internal_id":               pool.InternalID,
		"machine_manager_id":        pool.MachineManager.ID,
		"high_availability_enabled": pool.HighAvailabilityEnabled,
		"master":                    pool.Master,
		"hosts":                     pool.Hosts,
		"memory":                    memory,
		"cpu":                       cpu,
		"type":                      poolType,
	}
}
