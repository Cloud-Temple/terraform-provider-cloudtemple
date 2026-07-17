package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const PublicCloudVMInstanceId = "PUBLIC_CLOUD_VM_INSTANCE_ID"

func TestAccDataSourcePublicCloudVMInstance(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMInstance, os.Getenv(PublicCloudVMInstanceId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_instance.foo", "id", os.Getenv(PublicCloudVMInstanceId)),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance.foo", "status"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance.foo", "availability_zone.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance.foo", "disks_size_gb"),
				),
			},
			{
				Config:      testAccDataSourcePublicCloudVMInstanceMissing,
				ExpectError: regexp.MustCompile("failed to find VM instance with id"),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMInstance = `
data "cloudtemple_public_cloud_vm_instance" "foo" {
  id = "%s"
}
`

const testAccDataSourcePublicCloudVMInstanceMissing = `
data "cloudtemple_public_cloud_vm_instance" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
