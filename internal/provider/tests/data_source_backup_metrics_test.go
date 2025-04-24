package provider

import (
	"fmt"
	"strconv"
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
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier l'existence des sections principales
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "coverage.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "history.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "platform.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "platform_cpu.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "policies.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "virtual_machines.#"),

					// Vérifier les propriétés principales de chaque section
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "coverage.0.protected_resources"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "coverage.0.total_resources"),

					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "history.0.total_runs"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "history.0.success"),

					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "platform.0.version"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "platform.0.product"),

					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "platform_cpu.0.cpu_util"),

					// Vérifier que la liste des politiques n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_metrics.foo",
						"policies.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse policies count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected policies list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés de la première politique
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "policies.0.name"),

					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "virtual_machines.0.in_compute"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_metrics.foo", "virtual_machines.0.with_backup"),
				),
			},
		},
	})
}

const testAccDataSourceMetrics = `
data "cloudtemple_backup_metrics" "foo" {}
`
