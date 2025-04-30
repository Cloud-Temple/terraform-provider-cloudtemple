package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenToken convertit un objet Token en une map compatible avec le schéma Terraform
func FlattenToken(token *client.Token) map[string]interface{} {
	return map[string]interface{}{
		"id":              token.ID,
		"name":            token.Name,
		"roles":           token.Roles,
		"expiration_date": token.ExpirationDate,
	}
}
