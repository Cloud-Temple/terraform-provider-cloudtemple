package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceBackupSPPServers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupSPPServers,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des serveurs SPP n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_spp_servers.foo",
						"spp_servers.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse spp_servers count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected spp_servers list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_spp_servers.foo", "spp_servers.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_spp_servers.foo", "spp_servers.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_spp_servers.foo", "spp_servers.0.address"),
				),
			},
		},
	})
}

const testAccDataSourceBackupSPPServers = `
data "cloudtemple_backup_spp_servers" "foo" {}
`
