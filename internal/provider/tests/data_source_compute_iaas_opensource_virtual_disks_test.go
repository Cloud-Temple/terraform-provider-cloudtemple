package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOpenIaaSVirtualDisks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpenIaaSVirtualDisks,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des virtual_disks n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_virtual_disks.foo",
						"virtual_disks.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse virtual_disks count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected virtual_disks list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disks.foo", "virtual_disks.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disks.foo", "virtual_disks.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disks.foo", "virtual_disks.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disks.foo", "virtual_disks.0.storage_repository_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disks.foo", "virtual_disks.0.size"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSVirtualDisksWithFilter, os.Getenv(OpenIaaSVirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des virtual_disks n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_virtual_disks.filtered",
						"virtual_disks.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse virtual_disks count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected virtual_disks list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disks.filtered", "virtual_disks.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disks.filtered", "virtual_disks.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disks.filtered", "virtual_disks.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disks.filtered", "virtual_disks.0.storage_repository_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disks.filtered", "virtual_disks.0.size"),
				),
			},
		},
	})
}

const testAccDataSourceOpenIaaSVirtualDisks = `
data "cloudtemple_compute_iaas_opensource_virtual_disks" "foo" {}
`

const testAccDataSourceOpenIaaSVirtualDisksWithFilter = `
data "cloudtemple_compute_iaas_opensource_virtual_disks" "filtered" {
  virtual_machine_id = "%s"
}
`
