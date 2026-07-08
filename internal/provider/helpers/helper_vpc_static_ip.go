package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenStaticIP converts a StaticIP client object into the flat map consumed
// by the cloudtemple_vpc_static_ip / cloudtemple_vpc_static_ips datasource
// schemas.
//
// virtual_machine, network_adapter, resource_description and floating_ip are
// nullable in the API. Each nullable association is flattened to its id (and the
// floating IP also to its address); a null is flattened to an empty string so
// every Computed attribute is always set.
func FlattenStaticIP(staticIP *client.StaticIP) map[string]interface{} {
	result := map[string]interface{}{
		"id":                 staticIP.ID,
		"ip_address":         staticIP.IPAddress,
		"mac_address":        staticIP.MacAddress,
		"source":             staticIP.Source,
		"vpc_id":             staticIP.VPC.ID,
		"private_network_id": staticIP.PrivateNetwork.ID,
	}

	result["virtual_machine_id"] = ""
	if staticIP.VirtualMachine != nil {
		result["virtual_machine_id"] = staticIP.VirtualMachine.ID
	}

	result["network_adapter_id"] = ""
	if staticIP.NetworkAdapter != nil {
		result["network_adapter_id"] = staticIP.NetworkAdapter.ID
	}

	result["resource_description"] = ""
	if staticIP.ResourceDescription != nil {
		result["resource_description"] = *staticIP.ResourceDescription
	}

	result["floating_ip_id"] = ""
	result["floating_ip_address"] = ""
	if staticIP.FloatingIP != nil {
		result["floating_ip_id"] = staticIP.FloatingIP.ID
		result["floating_ip_address"] = staticIP.FloatingIP.IPAddress
	}

	return result
}
