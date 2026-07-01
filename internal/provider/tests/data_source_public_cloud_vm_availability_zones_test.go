package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMAvailabilityZones(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMAvailabilityZones,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_availability_zones.all", "availability_zones.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_availability_zones.all", "availability_zones.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_availability_zones.all", "availability_zones.0.region_id"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMAvailabilityZones = `
data "cloudtemple_public_cloud_vm_availability_zones" "all" {}
`
