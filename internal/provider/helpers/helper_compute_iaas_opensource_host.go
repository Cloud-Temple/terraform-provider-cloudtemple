package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSHost convertit un objet OpenIaaSHost en une map compatible avec le schéma Terraform
func FlattenOpenIaaSHost(host *client.OpenIaaSHost) map[string]interface{} {
	// Mapper le type de pool
	poolType := []map[string]interface{}{
		{
			"key":         host.Pool.Type.Key,
			"description": host.Pool.Type.Description,
		},
	}

	// Mapper le pool
	pool := []map[string]interface{}{
		{
			"id":   host.Pool.ID,
			"name": host.Pool.Name,
			"type": poolType,
		},
	}

	// Mapper les données de mise à jour
	updateData := []map[string]interface{}{
		{
			"maintenance_mode": host.UpdateData.MaintenanceMode,
			"status":           host.UpdateData.Status,
		},
	}

	// Mapper les métriques XOA
	xoa := []map[string]interface{}{
		{
			"version":   host.Metrics.XOA.Version,
			"full_name": host.Metrics.XOA.FullName,
			"build":     host.Metrics.XOA.Build,
		},
	}

	// Mapper les métriques de mémoire
	memory := []map[string]interface{}{
		{
			"usage": host.Metrics.Memory.Usage,
			"size":  host.Metrics.Memory.Size,
		},
	}

	// Mapper les métriques CPU
	cpu := []map[string]interface{}{
		{
			"sockets":    host.Metrics.Cpu.Sockets,
			"cores":      host.Metrics.Cpu.Cores,
			"model":      host.Metrics.Cpu.Model,
			"model_name": host.Metrics.Cpu.ModelName,
		},
	}

	// Mapper les métriques complètes
	metrics := []map[string]interface{}{
		{
			"xoa":    xoa,
			"memory": memory,
			"cpu":    cpu,
		},
	}

	return map[string]interface{}{
		"id":                 host.ID,
		"name":               host.Name,
		"internal_id":        host.InternalId,
		"machine_manager_id": host.MachineManager.ID,
		"pool":               pool,
		"master":             host.Master,
		"uptime":             host.Uptime,
		"power_state":        host.PowerState,
		"update_data":        updateData,
		"reboot_required":    host.RebootRequired,
		"virtual_machines":   host.VirtualMachines,
		"metrics":            metrics,
	}
}
