package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenUser convertit un objet User en une map compatible avec le schéma Terraform
func FlattenUser(user *client.User) map[string]interface{} {
	return map[string]interface{}{
		"id":             user.ID,
		"internal_id":    user.InternalID,
		"name":           user.Name,
		"type":           user.Type,
		"source":         user.Source,
		"source_id":      user.SourceID,
		"email_verified": user.EmailVerified,
		"email":          user.Email,
	}
}
