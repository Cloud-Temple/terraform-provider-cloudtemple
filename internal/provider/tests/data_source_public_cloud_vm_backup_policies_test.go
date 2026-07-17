package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMBackupPolicies(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMBackupPolicies,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_backup_policies.all", "backup_policies.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_backup_policies.all", "backup_policies.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_backup_policies.all", "backup_policies.0.retention"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMBackupPolicies = `
data "cloudtemple_public_cloud_vm_backup_policies" "all" {}
`
