package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceBackupSPPServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupSPPServer,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "id", "a3d46fb5-29af-4b98-a665-1e82a62fd6d3"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "name", "10"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "address", "10.1.11.32"),
				),
			},
			{
				Config: testAccDataSourceBackupSPPServerName,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "id", "a3d46fb5-29af-4b98-a665-1e82a62fd6d3"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "name", "10"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "address", "10.1.11.32"),
				),
			},
			{
				Config:      testAccDataSourceBackupSPPServerMissing,
				ExpectError: regexp.MustCompile("failed to find SPP server with id"),
			},
		},
	})
}

const testAccDataSourceBackupSPPServer = `
data "cloudtemple_backup_spp_server" "foo" {
  id = "a3d46fb5-29af-4b98-a665-1e82a62fd6d3"
}
`

const testAccDataSourceBackupSPPServerName = `
data "cloudtemple_backup_spp_server" "foo" {
  name = "10"
}
`

const testAccDataSourceBackupSPPServerMissing = `
data "cloudtemple_backup_spp_server" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
