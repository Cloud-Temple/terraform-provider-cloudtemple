package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenFloatingIP converts a FloatingIP client object into the flat map
// consumed by the cloudtemple_vpc_floating_ip / cloudtemple_vpc_floating_ips
// datasource schemas.
//
// static_ip, vpc and private_network are nullable in the API: they are
// populated only when the floating IP is bound to a static IP. Each association
// is flattened to its id (and the static IP also to its address); a null is
// flattened to an empty string so every Computed attribute is always set.
func FlattenFloatingIP(floatingIP *client.FloatingIP) map[string]interface{} {
	result := map[string]interface{}{
		"id":          floatingIP.ID,
		"ip_address":  floatingIP.IPAddress,
		"description": floatingIP.Description,
	}

	result["static_ip_id"] = ""
	result["static_ip_address"] = ""
	if floatingIP.StaticIP != nil {
		result["static_ip_id"] = floatingIP.StaticIP.ID
		result["static_ip_address"] = floatingIP.StaticIP.Address
	}

	result["vpc_id"] = ""
	if floatingIP.VPC != nil {
		result["vpc_id"] = floatingIP.VPC.ID
	}

	result["private_network_id"] = ""
	if floatingIP.PrivateNetwork != nil {
		result["private_network_id"] = floatingIP.PrivateNetwork.ID
	}

	return result
}
