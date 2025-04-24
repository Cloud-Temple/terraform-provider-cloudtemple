package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VCentersQty = "COMPUTE_VCENTER_QTY"
)

func TestAccDataSourceBackupVCenters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceBackupVCenters, os.Getenv(SppServerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des serveurs vCenter n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_vcenters.foo",
						"vcenters.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse vcenters count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected vcenters list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_vcenters.foo", "vcenters.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_vcenters.foo", "vcenters.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_vcenters.foo", "vcenters.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_vcenters.foo", "vcenters.0.instance_id"),
				),
			},
		},
	})
}

const testAccDataSourceBackupVCenters = `
data "cloudtemple_backup_vcenters" "foo" {
  spp_server_id = "%s"
}
`
