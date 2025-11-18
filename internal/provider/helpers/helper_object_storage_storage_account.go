package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenStorageAccount convertit un objet StorageAccount en une map compatible avec le schéma Terraform
func FlattenStorageAccount(account *client.StorageAccount) map[string]interface{} {
	tags := make([]map[string]interface{}, len(account.Tags))
	for i, tag := range account.Tags {
		tags[i] = map[string]interface{}{
			"key":   tag.Key,
			"value": tag.Value,
		}
	}

	return map[string]interface{}{
		"id":            account.ID,
		"name":          account.Name,
		"access_key_id": account.AccessKeyID,
		"arn":           account.ARN,
		"create_date":   account.CreateDate,
		"path":          account.Path,
		"tags":          tags,
	}
}
