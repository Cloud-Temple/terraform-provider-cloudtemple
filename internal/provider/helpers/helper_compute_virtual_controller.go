package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenVirtualController convertit un objet VirtualController en une map compatible avec le schéma Terraform
func FlattenVirtualController(controller *client.VirtualController) map[string]interface{} {
	return map[string]interface{}{
		"virtual_machine_id": controller.VirtualMachineId,
		"hot_add_remove":     controller.HotAddRemove,
		"type":               controller.Type,
		"sub_type":           controller.SubType,
		"label":              controller.Label,
		"summary":            controller.Summary,
		"virtual_disks":      controller.VirtualDisks,
	}
}
