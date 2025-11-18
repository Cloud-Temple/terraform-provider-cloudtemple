package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenACL convertit un objet ACL en une map compatible avec le schéma Terraform
func FlattenACL(acl *client.ACL) map[string]interface{} {
	return map[string]interface{}{
		"id":   acl.ID,
		"name": acl.Name,
		"role": acl.Role,
	}
}
