package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceBackupSLAPolicies(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupSLAPolicies,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policies.foo", "sla_policies.#", "10"),
				),
			},
		},
	})
}

const testAccDataSourceBackupSLAPolicies = `
data "cloudtemple_backup_sla_policies" "foo" {}
`
