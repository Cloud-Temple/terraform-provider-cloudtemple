package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMInstanceFamilyId   = "PUBLIC_CLOUD_VM_INSTANCE_FAMILY_ID"
	PublicCloudVMInstanceFamilyName = "PUBLIC_CLOUD_VM_INSTANCE_FAMILY_NAME"
)

func TestAccDataSourcePublicCloudVMInstanceFamily(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMInstanceFamily, os.Getenv(PublicCloudVMInstanceFamilyId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance_family.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance_family.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance_family.foo", "vcpu_max"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMInstanceFamilyName, os.Getenv(PublicCloudVMInstanceFamilyName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance_family.foo", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_instance_family.foo", "name", os.Getenv(PublicCloudVMInstanceFamilyName)),
				),
			},
			{
				Config:      fmt.Sprintf(testAccDataSourcePublicCloudVMInstanceFamilyConflict, os.Getenv(PublicCloudVMInstanceFamilyId), os.Getenv(PublicCloudVMInstanceFamilyName)),
				ExpectError: regexp.MustCompile(`"id": conflicts with name`),
			},
			{
				Config:      testAccDataSourcePublicCloudVMInstanceFamilyMissing,
				ExpectError: regexp.MustCompile("failed to find instance family with id"),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMInstanceFamily = `
data "cloudtemple_public_cloud_vm_instance_family" "foo" {
  id = "%s"
}
`

const testAccDataSourcePublicCloudVMInstanceFamilyName = `
data "cloudtemple_public_cloud_vm_instance_family" "foo" {
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMInstanceFamilyConflict = `
data "cloudtemple_public_cloud_vm_instance_family" "foo" {
  id   = "%s"
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMInstanceFamilyMissing = `
data "cloudtemple_public_cloud_vm_instance_family" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
