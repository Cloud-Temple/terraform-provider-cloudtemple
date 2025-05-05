package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceHosts(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceHosts,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_hosts.foo",
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
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_hosts.foo", "hosts.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_hosts.foo", "hosts.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_hosts.foo", "hosts.0.moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_hosts.foo", "hosts.0.machine_manager_id"),
				),
			},
		},
	})
}

const testAccDataSourceHosts = `
data "cloudtemple_compute_hosts" "foo" {}
`
