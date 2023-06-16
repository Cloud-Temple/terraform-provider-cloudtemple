package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	SnapShotQty = "COMPUTE_SNAPSHOT_QTY"
)

func TestAccDataSourceSnapshots(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceSnapshots, os.Getenv(VirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_snapshots.foo", "snapshots.#", os.Getenv(SnapShotQty)),
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
