package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSNetwork convertit un objet OpenIaaSNetwork en une map compatible avec le schéma Terraform
func FlattenOpenIaaSNetwork(network *client.OpenIaaSNetwork) map[string]interface{} {
	return map[string]interface{}{
		"id":                            network.ID,
		"name":                          network.Name,
		"internal_id":                   network.InternalID,
		"machine_manager_id":            network.MachineManager.ID,
		"pool_id":                       network.Pool.ID,
		"maximum_transmission_unit":     network.MaximumTransmissionUnit,
		"network_adapters":              network.NetworkAdapters,
		"network_block_device":          network.NetworkBlockDevice,
		"insecure_network_block_device": network.InsecureNetworkBlockDevice,
	}
}
