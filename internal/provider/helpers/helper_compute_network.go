package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenNetwork convertit un objet Network en une map compatible avec le schéma Terraform
func FlattenNetwork(network *client.Network) map[string]interface{} {
	return map[string]interface{}{
		"id":                      network.ID,
		"name":                    network.Name,
		"moref":                   network.Moref,
		"machine_manager_id":      network.MachineManager.ID,
		"virtual_machines_number": network.VirtualMachinesNumber,
		"host_number":             network.HostNumber,
		"host_names":              network.HostNames,
	}
}
