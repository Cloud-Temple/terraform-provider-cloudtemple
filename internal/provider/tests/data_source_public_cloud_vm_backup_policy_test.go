package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMBackupPolicyName = "PUBLIC_CLOUD_VM_BACKUP_POLICY_NAME"
)

func TestAccDataSourcePublicCloudVMBackupPolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMBackupPolicyName, os.Getenv(PublicCloudVMBackupPolicyName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_backup_policy.foo", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_backup_policy.foo", "name", os.Getenv(PublicCloudVMBackupPolicyName)),
				),
			},
			{
				Config:      testAccDataSourcePublicCloudVMBackupPolicyMissing,
				ExpectError: regexp.MustCompile("failed to find backup policy with id"),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMBackupPolicyName = `
data "cloudtemple_public_cloud_vm_backup_policy" "foo" {
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMBackupPolicyMissing = `
data "cloudtemple_public_cloud_vm_backup_policy" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
