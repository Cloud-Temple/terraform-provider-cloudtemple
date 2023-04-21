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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_vcenters.foo", "vcenters.#", "2"),
				),
			},
		},
	})
}

const testAccDataSourceBackupVCenters = `
data "cloudtemple_backup_vcenters" "foo" {
  spp_server_id = "a34d230c-dd0f-4fa9-a099-bec7d8609bd4"
}
`
