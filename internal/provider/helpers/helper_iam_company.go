package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenCompany convertit un objet Company en une map compatible avec le schéma Terraform
func FlattenCompany(company *client.Company) map[string]interface{} {
	return map[string]interface{}{
		"id":   company.ID,
		"name": company.Name,
	}
}
