package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSnapshots(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceSnapshots, os.Getenv(VirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des snapshots n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_snapshots.foo",
						"snapshots.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse snapshots count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected snapshots list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_snapshots.foo", "snapshots.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_snapshots.foo", "snapshots.0.virtual_machine_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_snapshots.foo", "snapshots.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_snapshots.foo", "snapshots.0.create_time"),
				),
			},
			{
				Config: testAccDataSourceSnapshotsMissing,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_snapshots.foo", "snapshots.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceSnapshots = `
data "cloudtemple_compute_snapshots" "foo" {
  virtual_machine_id = "%s"
}
`

const testAccDataSourceSnapshotsMissing = `
data "cloudtemple_compute_snapshots" "foo" {
  virtual_machine_id = "12345678-1234-5678-1234-567812345678"
}
`
