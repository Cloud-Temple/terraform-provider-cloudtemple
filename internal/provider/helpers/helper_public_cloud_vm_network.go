package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMNetwork maps a client.PublicCloudVMNetwork to the flat
// snake_case map consumed by both the single and list datasources. The `vpc`
// block is always emitted — empty for a Private Backbone network, populated for
// a VPC-backed one — so a config can use length(vpc) as the discriminator.
func FlattenPublicCloudVMNetwork(network *client.PublicCloudVMNetwork) map[string]interface{} {
	vpc := []map[string]interface{}{}
	if network.VPC != nil {
		privateNetwork := []map[string]interface{}{}
		if network.VPC.PrivateNetwork != nil {
			privateNetwork = append(privateNetwork, map[string]interface{}{
				"id":   network.VPC.PrivateNetwork.ID,
				"name": network.VPC.PrivateNetwork.Name,
			})
		}
		vpc = append(vpc, map[string]interface{}{
			"id":              network.VPC.ID,
			"name":            network.VPC.Name,
			"private_network": privateNetwork,
		})
	}
	return map[string]interface{}{
		"id":   network.ID,
		"name": network.Name,
		"vpc":  vpc,
	}
}
