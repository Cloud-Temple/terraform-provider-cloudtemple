package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenVPC converts a VPC client object into the flat map consumed by the
// cloudtemple_vpc_vpc / cloudtemple_vpc_vpcs datasource schemas.
//
// internet_ip is nullable in the API; a null is flattened to an empty string so
// the attribute is always set (the schema declares it Computed/string).
func FlattenVPC(vpc *client.VPC) map[string]interface{} {
	internetIP := ""
	if vpc.InternetIP != nil {
		internetIP = *vpc.InternetIP
	}

	return map[string]interface{}{
		"id":                    vpc.ID,
		"name":                  vpc.Name,
		"internet_ip":           internetIP,
		"private_network_count": vpc.PrivateNetworkCount,
		"static_ip_count":       vpc.StaticIPCount,
		"floating_ip_count":     vpc.FloatingIPCount,
	}
}
