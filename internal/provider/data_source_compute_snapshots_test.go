package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSnapshots(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceSnapshots,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_snapshots.foo", "snapshots.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceSnapshots = `
data "cloudtemple_compute_snapshots" "foo" {
  virtual_machine_id = "de2b8b80-8b90-414a-bc33-e12f61a4c05c"
}
`
