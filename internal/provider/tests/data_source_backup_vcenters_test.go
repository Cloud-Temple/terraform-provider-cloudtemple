package provider

import (
	"fmt"
	"os"
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
					resource.TestCheckResourceAttr("data.cloudtemple_backup_vcenters.foo", "vcenters.#", os.Getenv(VCentersQty)),
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
