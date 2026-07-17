package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMAvailabilityZoneId   = "PUBLIC_CLOUD_VM_AZ_ID"
	PublicCloudVMAvailabilityZoneName = "PUBLIC_CLOUD_VM_AZ_NAME"
)

func TestAccDataSourcePublicCloudVMAvailabilityZone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMAvailabilityZone, os.Getenv(PublicCloudVMAvailabilityZoneId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_availability_zone.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_availability_zone.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_availability_zone.foo", "region_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMAvailabilityZoneName, os.Getenv(PublicCloudVMAvailabilityZoneName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_availability_zone.foo", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_availability_zone.foo", "name", os.Getenv(PublicCloudVMAvailabilityZoneName)),
				),
			},
			{
				Config:      fmt.Sprintf(testAccDataSourcePublicCloudVMAvailabilityZoneConflict, os.Getenv(PublicCloudVMAvailabilityZoneId), os.Getenv(PublicCloudVMAvailabilityZoneName)),
				ExpectError: regexp.MustCompile(`"id": conflicts with name`),
			},
			{
				Config:      testAccDataSourcePublicCloudVMAvailabilityZoneMissing,
				ExpectError: regexp.MustCompile("failed to find availability zone with id"),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMAvailabilityZone = `
data "cloudtemple_public_cloud_vm_availability_zone" "foo" {
  id = "%s"
}
`

const testAccDataSourcePublicCloudVMAvailabilityZoneName = `
data "cloudtemple_public_cloud_vm_availability_zone" "foo" {
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMAvailabilityZoneConflict = `
data "cloudtemple_public_cloud_vm_availability_zone" "foo" {
  id   = "%s"
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMAvailabilityZoneMissing = `
data "cloudtemple_public_cloud_vm_availability_zone" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
