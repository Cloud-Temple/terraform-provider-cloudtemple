package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenStaticIP converts a StaticIP client object to a Terraform-compatible map
func FlattenStaticIP(staticIP *client.StaticIP) map[string]interface{} {
	result := map[string]interface{}{
		"id":                 staticIP.ID,
		"ip_address":         staticIP.IPAddress,
		"mac_address":        staticIP.MacAddress,
		"source":             staticIP.Source,
		"vpc_id":             staticIP.VPC.ID,
		"private_network_id": staticIP.PrivateNetwork.ID,
	}

	// Handle nullable virtual_machine
	if staticIP.VirtualMachine != nil {
		result["virtual_machine_id"] = staticIP.VirtualMachine.ID
	} else {
		result["virtual_machine_id"] = ""
	}

	// Handle nullable network_adapter
	if staticIP.NetworkAdapter != nil {
		result["network_adapter_id"] = staticIP.NetworkAdapter.ID
	} else {
		result["network_adapter_id"] = ""
	}

	// Handle nullable resource_description
	if staticIP.ResourceDescription != nil {
		result["resource_description"] = *staticIP.ResourceDescription
	} else {
		result["resource_description"] = ""
	}

	// Handle nullable floating_ip
	if staticIP.FloatingIP != nil {
		result["floating_ip_id"] = staticIP.FloatingIP.ID
		result["floating_ip_address"] = staticIP.FloatingIP.IPAddress
	} else {
		result["floating_ip_id"] = ""
		result["floating_ip_address"] = ""
	}

	return result
}
