package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenGuestOperatingSystem convertit un objet GuestOperatingSystem en une map compatible avec le schéma Terraform
func FlattenGuestOperatingSystem(guestOS *client.GuestOperatingSystem) map[string]interface{} {
	return map[string]interface{}{
		"moref":     guestOS.Moref,
		"family":    guestOS.Family,
		"full_name": guestOS.FullName,
	}
}
