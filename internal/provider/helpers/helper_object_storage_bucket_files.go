package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBucketFile convertit un objet BucketFile en une map compatible avec le schéma Terraform
func FlattenBucketFile(file *client.BucketFile) map[string]interface{} {
	tags := make([]map[string]interface{}, len(file.Tags))
	for i, tag := range file.Tags {
		tags[i] = map[string]interface{}{
			"key":   tag.Key,
			"value": tag.Value,
		}
	}

	versions := make([]map[string]interface{}, len(file.Versions))
	for i, version := range file.Versions {
		versions[i] = map[string]interface{}{
			"version_id":       version.VersionID,
			"is_latest":        version.IsLatest,
			"last_modified":    version.LastModified,
			"size":             version.Size,
			"is_delete_marker": version.IsDeleteMarker,
		}
	}

	return map[string]interface{}{
		"key":           file.Key,
		"last_modified": file.LastModified,
		"size":          file.Size,
		"tags":          tags,
		"versions":      versions,
	}
}
