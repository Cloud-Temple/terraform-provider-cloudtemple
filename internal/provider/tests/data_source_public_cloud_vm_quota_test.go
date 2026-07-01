package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMQuota(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMQuota,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_quota.current", "vcpu_limit"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_quota.current", "ram_limit_mb"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_quota.current", "storage_limit_gb"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMQuota = `
data "cloudtemple_public_cloud_vm_quota" "current" {}
`
