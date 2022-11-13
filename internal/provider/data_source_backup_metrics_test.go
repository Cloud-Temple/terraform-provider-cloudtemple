package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceMetrics(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceMetrics,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_metrics.foo", "coverage.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_metrics.foo", "history.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_metrics.foo", "platform.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_metrics.foo", "platform_cpu.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_metrics.foo", "policies.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_metrics.foo", "virtual_machines.#", "1"),
				),
			},
		},
	})
}

const testAccDataSourceMetrics = `
data "cloudtemple_backup_metrics" "foo" {}
`
