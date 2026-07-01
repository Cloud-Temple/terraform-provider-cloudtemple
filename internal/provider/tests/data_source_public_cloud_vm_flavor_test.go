package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMFlavorName = "PUBLIC_CLOUD_VM_FLAVOR_NAME"
)

func TestAccDataSourcePublicCloudVMFlavor(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMFlavorName, os.Getenv(PublicCloudVMFlavorName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_flavor.foo", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_flavor.foo", "name", os.Getenv(PublicCloudVMFlavorName)),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_flavor.foo", "vcpu"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_flavor.foo", "ram_gb"),
				),
			},
			{
				Config:      testAccDataSourcePublicCloudVMFlavorMissing,
				ExpectError: regexp.MustCompile("failed to find flavor with id"),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMFlavorName = `
data "cloudtemple_public_cloud_vm_flavor" "foo" {
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMFlavorMissing = `
data "cloudtemple_public_cloud_vm_flavor" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
