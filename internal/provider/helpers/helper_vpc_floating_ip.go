package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenFloatingIP converts a FloatingIP client object to a Terraform-compatible map
func FlattenFloatingIP(floatingIP *client.FloatingIP) map[string]interface{} {
	result := map[string]interface{}{
		"id":          floatingIP.ID,
		"ip_address":  floatingIP.IPAddress,
		"description": floatingIP.Description,
	}

	// Handle nullable static_ip
	if floatingIP.StaticIP != nil {
		result["static_ip_id"] = floatingIP.StaticIP.ID
		result["static_ip_address"] = floatingIP.StaticIP.Address
	} else {
		result["static_ip_id"] = ""
		result["static_ip_address"] = ""
	}

	// Handle nullable vpc
	if floatingIP.VPC != nil {
		result["vpc_id"] = floatingIP.VPC.ID
	} else {
		result["vpc_id"] = ""
	}

	// Handle nullable private_network
	if floatingIP.PrivateNetwork != nil {
		result["private_network_id"] = floatingIP.PrivateNetwork.ID
	} else {
		result["private_network_id"] = ""
	}

	return result
}
