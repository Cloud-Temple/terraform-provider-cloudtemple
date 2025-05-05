package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOpenIaaSAvailabilityZones(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpenIaaSAvailabilityZones,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des availability_zones n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_availability_zones.foo",
						"availability_zones.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse availability_zones count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected availability_zones list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zones.foo", "availability_zones.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zones.foo", "availability_zones.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zones.foo", "availability_zones.0.os_name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zones.foo", "availability_zones.0.os_version"),
				),
			},
		},
	})
}

const testAccDataSourceOpenIaaSAvailabilityZones = `
data "cloudtemple_compute_iaas_opensource_availability_zones" "foo" {}
`
