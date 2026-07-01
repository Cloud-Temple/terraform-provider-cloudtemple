package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMNetwork maps a client.PublicCloudVMNetwork to the flat
// snake_case map consumed by both the single and list datasources. The live
// catalogue object is minimal ({id, name}); the spec's optional `vpc` block is
// not exposed yet (never observed on the live platform).
func FlattenPublicCloudVMNetwork(network *client.PublicCloudVMNetwork) map[string]interface{} {
	return map[string]interface{}{
		"id":   network.ID,
		"name": network.Name,
	}
}
