package helpers

import "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"

func FlattenObjectStorageRole(role *client.ObjectStorageRole) map[string]interface{} {
	return map[string]interface{}{
		"id":          role.ID,
		"name":        role.Name,
		"description": role.Description,
		"permissions": role.Permissions,
	}
}
