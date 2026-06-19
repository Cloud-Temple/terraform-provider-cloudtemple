package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSNetworkAdapter convertit un objet OpenIaaSNetworkAdapter en une map compatible avec le schéma Terraform.
//
// The VPC association is nullable in the API (the `vpc` object is present only
// when the adapter is on a VPC network). The VPC/static-IP/private-network
// attributes are flattened to FLAT Computed strings, and an absent association
// collapses to "" so every Computed key is always set (never nil, never a
// panic, #238). The NON-deprecated nested privateNetwork{id,name} is used; the
// deprecated top-level privateNetworkId/Name are ignored.
func FlattenOpenIaaSNetworkAdapter(adapter *client.OpenIaaSNetworkAdapter) map[string]interface{} {
	result := map[string]interface{}{
		"name":               adapter.Name,
		"internal_id":        adapter.InternalID,
		"virtual_machine_id": adapter.VirtualMachineID,
		"mac_address":        adapter.MacAddress,
		"mtu":                adapter.MTU,
		"attached":           adapter.Attached,
		"tx_checksumming":    adapter.TxChecksumming,
		"network_id":         adapter.Network.ID,
		"machine_manager_id": adapter.MachineManager.ID,
		"ipv4_address":       adapter.IPv4Address,
		"ipv6_address":       adapter.IPv6Address,
	}

	result["vpc_id"] = ""
	result["vpc_name"] = ""
	result["private_network_id"] = ""
	result["private_network_name"] = ""
	result["static_ip_address"] = ""
	if adapter.VPC != nil {
		result["vpc_id"] = adapter.VPC.ID
		result["vpc_name"] = adapter.VPC.Name
		result["private_network_id"] = adapter.VPC.PrivateNetwork.ID
		result["private_network_name"] = adapter.VPC.PrivateNetwork.Name
		result["static_ip_address"] = adapter.VPC.StaticIPAddress
	}

	return result
}
