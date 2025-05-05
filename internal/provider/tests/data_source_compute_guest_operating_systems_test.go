package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceGuestOperatingSystems(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceGuestOperatingSystems, os.Getenv(HostClusterId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des systèmes d'exploitation invités n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_guest_operating_systems.foo",
						"guest_operating_systems.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse guest_operating_systems count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected guest_operating_systems list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_guest_operating_systems.foo", "guest_operating_systems.0.moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_guest_operating_systems.foo", "guest_operating_systems.0.family"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_guest_operating_systems.foo", "guest_operating_systems.0.full_name"),
				),
			},
			{
				Config: testAccDataSourceGuestOperatingSystemsMissing,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_guest_operating_systems.foo", "guest_operating_systems.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceGuestOperatingSystems = `
data "cloudtemple_compute_guest_operating_systems" "foo" {
  host_cluster_id = "%s"
}
`

const testAccDataSourceGuestOperatingSystemsMissing = `
data "cloudtemple_compute_guest_operating_systems" "foo" {
  host_cluster_id = "f7336dc8-7a91-461f-933d-3642aa415446"
}
`
