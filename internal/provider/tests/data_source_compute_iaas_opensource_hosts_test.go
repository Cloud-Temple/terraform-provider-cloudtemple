package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOpenIaaSHosts(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpenIaaSHosts,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des hosts n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_hosts.foo",
						"hosts.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse hosts count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected hosts list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_hosts.foo", "hosts.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_hosts.foo", "hosts.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_hosts.foo", "hosts.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_hosts.foo", "hosts.0.machine_manager_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSHostsWithFilter, os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des hosts n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_hosts.filtered",
						"hosts.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse hosts count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected hosts list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_hosts.filtered", "hosts.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_hosts.filtered", "hosts.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_hosts.filtered", "hosts.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_hosts.filtered", "hosts.0.machine_manager_id"),
				),
			},
		},
	})
}

const testAccDataSourceOpenIaaSHosts = `
data "cloudtemple_compute_iaas_opensource_hosts" "foo" {}
`

const testAccDataSourceOpenIaaSHostsWithFilter = `
data "cloudtemple_compute_iaas_opensource_hosts" "filtered" {
  machine_manager_id = "%s"
}
`
