package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceBackupVCenters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupVCenters,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_vcenters.foo", "vcenters.#", "1"),
				),
			},
		},
	})
}

const testAccDataSourceBackupVCenters = `
data "cloudtemple_backup_vcenters" "foo" {
  spp_server_id = "a3d46fb5-29af-4b98-a665-1e82a62fd6d3"
}
`
