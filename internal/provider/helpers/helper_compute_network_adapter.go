package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenNetworkAdapter convertit un objet NetworkAdapter en une map compatible avec le schéma Terraform
func FlattenNetworkAdapter(adapter *client.NetworkAdapter) map[string]interface{} {
	return map[string]interface{}{
		"virtual_machine_id": adapter.VirtualMachineId,
		"name":               adapter.Name,
		"network_id":         adapter.NetworkId,
		"type":               adapter.Type,
		"mac_type":           adapter.MacType,
		"mac_address":        adapter.MacAddress,
		"connected":          adapter.Connected,
		"auto_connect":       adapter.AutoConnect,
	}
}
