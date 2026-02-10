package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenHost convertit un objet Host en une map compatible avec le schéma Terraform
func FlattenHost(host *client.Host) map[string]interface{} {
	// Mapper les métriques ESX
	esx := []map[string]interface{}{
		{
			"version":   host.Metrics.ESX.Version,
			"build":     host.Metrics.ESX.Build,
			"full_name": host.Metrics.ESX.FullName,
		},
	}

	// Mapper les métriques CPU
	cpu := []map[string]interface{}{
		{
			"overall_cpu_usage": host.Metrics.CPU.OverallCPUUsage,
			"cpu_mhz":           host.Metrics.CPU.CPUMhz,
			"cpu_cores":         host.Metrics.CPU.CPUCores,
			"cpu_threads":       host.Metrics.CPU.CPUThreads,
		},
	}

	// Mapper les métriques mémoire
	memory := []map[string]interface{}{
		{
			"memory_size":  host.Metrics.Memory.MemorySize,
			"memory_usage": host.Metrics.Memory.MemoryUsage,
		},
	}

	// Mapper les métriques complètes
	metrics := []map[string]interface{}{
		{
			"esx":                esx,
			"cpu":                cpu,
			"memory":             memory,
			"maintenance_status": host.Metrics.MaintenanceMode, // Deprecated: use maintenance_mode
			"maintenance_mode":   host.Metrics.MaintenanceMode,
			"uptime":             host.Metrics.Uptime,
			"connected":          host.Metrics.Connected,
		},
	}

	// Mapper les machines virtuelles
	virtualMachines := make([]map[string]interface{}, len(host.VirtualMachines))
	for i, vm := range host.VirtualMachines {
		virtualMachines[i] = map[string]interface{}{
			"id":   vm.ID,
			"type": vm.Type,
		}
	}

	return map[string]interface{}{
		"id":                 host.ID,
		"name":               host.Name,
		"moref":              host.Moref,
		"machine_manager_id": host.MachineManager.ID,
		"metrics":            metrics,
		"virtual_machines":   virtualMachines,
	}
}
