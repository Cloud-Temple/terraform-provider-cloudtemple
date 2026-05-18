package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPrivateNetwork converts a PrivateNetwork client object to a Terraform-compatible map
func FlattenPrivateNetwork(network *client.PrivateNetwork) map[string]interface{} {
	result := map[string]interface{}{
		"id":              network.ID,
		"ip_address":      network.IPAddress,
		"vlan_id":         network.VlanID,
		"static_ip_count": network.StaticIPCount,
		"vpc_id":          network.VPC.ID,
	}

	// Handle nullable name
	if network.Name != nil {
		result["name"] = *network.Name
	} else {
		result["name"] = ""
	}

	return result
}
