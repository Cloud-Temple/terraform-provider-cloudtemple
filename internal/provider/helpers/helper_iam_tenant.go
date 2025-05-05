package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenTenant convertit un objet Tenant en une map compatible avec le schéma Terraform
func FlattenTenant(tenant *client.Tenant) map[string]interface{} {
	return map[string]interface{}{
		"id":         tenant.ID,
		"name":       tenant.Name,
		"snc":        tenant.SNC,
		"company_id": tenant.CompanyID,
	}
}
