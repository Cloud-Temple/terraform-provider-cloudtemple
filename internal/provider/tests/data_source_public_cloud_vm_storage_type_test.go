package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMStorageTypeName = "PUBLIC_CLOUD_VM_STORAGE_TYPE_NAME"
)

func TestAccDataSourcePublicCloudVMStorageType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMStorageTypeName, os.Getenv(PublicCloudVMStorageTypeName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_type.foo", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_storage_type.foo", "name", os.Getenv(PublicCloudVMStorageTypeName)),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_type.foo", "max_size_gb"),
				),
			},
			{
				Config:      testAccDataSourcePublicCloudVMStorageTypeMissing,
				ExpectError: regexp.MustCompile("failed to find storage type with id"),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMStorageTypeName = `
data "cloudtemple_public_cloud_vm_storage_type" "foo" {
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMStorageTypeMissing = `
data "cloudtemple_public_cloud_vm_storage_type" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
