package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenVirtualSwitch convertit un objet VirtualSwitch en une map compatible avec le schéma Terraform
func FlattenBucket(bucket *client.Bucket) map[string]interface{} {
	return map[string]interface{}{
		"id":                    bucket.ID,
		"name":                  bucket.Name,
		"namespace":             bucket.Namespace,
		"retention_period":      bucket.RetentionPeriod,
		"versioning":            bucket.Versioning,
		"endpoint":              bucket.Endpoint,
		"total_size":            bucket.TotalSize,
		"total_size_unit":       bucket.TotalSizeUnit,
		"total_objects":         bucket.TotalObjects,
		"total_objects_deleted": bucket.TotalObjectsDeleted,
		"total_size_deleted":    bucket.TotalSizeDeleted,
	}
}
