package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenVPC converts a VPC client object to a Terraform-compatible map
func FlattenVPC(vpc *client.VPC) map[string]interface{} {
	result := map[string]interface{}{
		"id":                    vpc.ID,
		"name":                  vpc.Name,
		"private_network_count": vpc.PrivateNetworkCount,
		"static_ip_count":       vpc.StaticIPCount,
		"floating_ip_count":     vpc.FloatingIPCount,
		"internet_ip":           vpc.InternetIP,
	}

	// // Handle nullable internet_ip
	// if vpc.InternetIP != nil {
	// 	result["internet_ip"] = *vpc.InternetIP
	// } else {
	// 	result["internet_ip"] = ""
	// }

	return result
}
