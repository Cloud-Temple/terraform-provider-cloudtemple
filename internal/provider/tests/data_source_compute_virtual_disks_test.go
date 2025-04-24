package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VistualDisksQty = "COMPUTE_VIRTUAL_DISK_QTY"
)

func TestAccDataSourceVirtualDisks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualDisks, os.Getenv(VirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des disques virtuels n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_virtual_disks.foo",
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
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disks.foo", "virtual_disks.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disks.foo", "virtual_disks.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disks.foo", "virtual_disks.0.virtual_machine_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disks.foo", "virtual_disks.0.capacity"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disks.foo", "virtual_disks.0.machine_manager_id"),
				),
			},
			{
				Config: testAccDataSourceVirtualDisksMissing,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disks.foo", "virtual_disks.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceVirtualDisks = `
data "cloudtemple_compute_virtual_disks" "foo" {
  virtual_machine_id = "%s"
}
`

const testAccDataSourceVirtualDisksMissing = `
data "cloudtemple_compute_virtual_disks" "foo" {
  virtual_machine_id = "12345678-1234-5678-1234-567812345678"
}
`
