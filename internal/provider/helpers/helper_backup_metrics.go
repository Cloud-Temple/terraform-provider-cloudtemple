package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBackupMetricsCoverage convertit un objet BackupMetricsCoverage en une map compatible avec le schéma Terraform
func FlattenBackupMetricsCoverage(coverage *client.BackupMetricsCoverage) map[string]interface{} {
	return map[string]interface{}{
		"failed_resources":      coverage.FailedResources,
		"protected_resources":   coverage.ProtectedResources,
		"unprotected_resources": coverage.UnprotectedResources,
		"total_resources":       coverage.TotalResources,
	}
}

// FlattenBackupMetricsHistory convertit un objet BackupMetricsHistory en une map compatible avec le schéma Terraform
func FlattenBackupMetricsHistory(history *client.BackupMetricsHistory) map[string]interface{} {
	return map[string]interface{}{
		"total_runs":     history.TotalRuns,
		"sucess_percent": int(history.SucessPercent), // Conversion en int pour correspondre au schéma
		"failed":         history.Failed,
		"warning":        history.Warning,
		"success":        history.Success,
		"running":        history.Running,
	}
}

// FlattenBackupMetricsPlatform convertit un objet BackupMetricsPlatform en une map compatible avec le schéma Terraform
func FlattenBackupMetricsPlatform(platform *client.BackupMetricsPlatform) map[string]interface{} {
	return map[string]interface{}{
		"version":         platform.Version,
		"build":           platform.Build,
		"date":            platform.Date,
		"product":         platform.Product,
		"epoch":           platform.Epoch,
		"deployment_type": platform.DeploymentType,
	}
}

// FlattenBackupMetricsPlatformCPU convertit un objet BackupMetricsPlatformCPU en une map compatible avec le schéma Terraform
func FlattenBackupMetricsPlatformCPU(platformCPU *client.BackupMetricsPlatformCPU) map[string]interface{} {
	return map[string]interface{}{
		"cpu_util": platformCPU.CPUUtil,
	}
}

// FlattenBackupMetricsPolicy convertit un objet BackupMetricsPolicies en une map compatible avec le schéma Terraform
func FlattenBackupMetricsPolicy(policy *client.BackupMetricsPolicies) map[string]interface{} {
	return map[string]interface{}{
		"name":                   policy.Name,
		"trigger_type":           policy.TriggerType,
		"number_of_protected_vm": policy.NumberOfProtectedVM,
	}
}

// FlattenBackupMetricsVirtualMachines convertit un objet BackupMetricsVirtualMachines en une map compatible avec le schéma Terraform
func FlattenBackupMetricsVirtualMachines(virtualMachines *client.BackupMetricsVirtualMachines) map[string]interface{} {
	return map[string]interface{}{
		"in_spp":                virtualMachines.InSPP,
		"in_compute":            virtualMachines.InCompute,
		"with_backup":           virtualMachines.WithBackup,
		"in_sla":                virtualMachines.InSLA,
		"in_offloading_sla":     virtualMachines.InOffloadingSLA,
		"tsm_offloading_factor": virtualMachines.TSMOffloadingFactor,
	}
}
