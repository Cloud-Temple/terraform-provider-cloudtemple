package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSSnapshotId = "COMPUTE_IAAS_OPENSOURCE_SNAPSHOT_ID"
)

func TestAccDataSourceOpenIaaSSnapshot(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSSnapshot, os.Getenv(OpenIaaSSnapshotId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_snapshot.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_snapshot.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_snapshot.foo", "description"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_snapshot.foo", "virtual_machine_id"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSSnapshotMissing,
				ExpectError: regexp.MustCompile("failed to find snapshot with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSSnapshot = `
data "cloudtemple_compute_iaas_opensource_snapshot" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSSnapshotMissing = `
data "cloudtemple_compute_iaas_opensource_snapshot" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
