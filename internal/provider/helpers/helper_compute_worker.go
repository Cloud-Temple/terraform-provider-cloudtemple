package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenWorker convertit un objet Worker en une map compatible avec le schéma Terraform
func FlattenWorker(worker *client.Worker) map[string]interface{} {
	return map[string]interface{}{
		"id":                      worker.ID,
		"name":                    worker.Name,
		"full_name":               worker.FullName,
		"vendor":                  worker.Vendor,
		"version":                 worker.Version,
		"build":                   worker.Build,
		"locale_version":          worker.LocaleVersion,
		"locale_build":            worker.LocaleBuild,
		"os_type":                 worker.OsType,
		"product_line_id":         worker.ProductLineID,
		"api_type":                worker.ApiType,
		"api_version":             worker.ApiVersion,
		"instance_uuid":           worker.InstanceUuid,
		"license_product_name":    worker.LicenseProductName,
		"license_product_version": worker.LicenseProductVersion,
		"tenant_id":               worker.TenantID,
		"tenant_name":             worker.TenantName,
	}
}
