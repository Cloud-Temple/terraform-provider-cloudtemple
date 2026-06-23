package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPrivateNetwork converts a PrivateNetwork client object into the flat
// map consumed by the cloudtemple_vpc_private_network /
// cloudtemple_vpc_private_networks datasource schemas.
//
// name is nullable in the API; a null is flattened to an empty string. The
// associated VPC is exposed as its id (vpc_id). gateway is intentionally absent:
// the API does not expose it yet.
func FlattenPrivateNetwork(network *client.PrivateNetwork) map[string]interface{} {
	name := ""
	if network.Name != nil {
		name = *network.Name
	}

	return map[string]interface{}{
		"id":              network.ID,
		"name":            name,
		"ip_address":      network.IPAddress,
		"vlan_id":         network.VlanID,
		"static_ip_count": network.StaticIPCount,
		"vpc_id":          network.VPC.ID,
	}
}
