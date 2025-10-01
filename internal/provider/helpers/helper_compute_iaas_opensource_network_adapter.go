package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSNetworkAdapter convertit un objet OpenIaaSNetworkAdapter en une map compatible avec le schéma Terraform
func FlattenOpenIaaSNetworkAdapter(adapter *client.OpenIaaSNetworkAdapter) map[string]interface{} {
	return map[string]interface{}{
		"name":               adapter.Name,
		"internal_id":        adapter.InternalID,
		"virtual_machine_id": adapter.VirtualMachineID,
		"mac_address":        adapter.MacAddress,
		"mtu":                adapter.MTU,
		"attached":           adapter.Attached,
		"tx_checksumming":    adapter.TxChecksumming,
		"network_id":         adapter.Network.ID,
		"machine_manager_id": adapter.MachineManager.ID,
	}
}
