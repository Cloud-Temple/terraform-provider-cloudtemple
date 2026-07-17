package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMSnapshot maps a client.PublicCloudVMSnapshot to the flat
// snake_case map consumed by the snapshots datasource.
func FlattenPublicCloudVMSnapshot(s *client.PublicCloudVMSnapshot) map[string]interface{} {
	return map[string]interface{}{
		"id":                 s.ID,
		"virtual_machine_id": s.VmID,
		"name":               s.Name,
		"status":             s.Status,
		"created_at":         s.CreatedAt,
	}
}
