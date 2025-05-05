package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenRole convertit un objet Role en une map compatible avec le schéma Terraform
func FlattenRole(role *client.Role) map[string]interface{} {
	return map[string]interface{}{
		"id":   role.ID,
		"name": role.Name,
	}
}
