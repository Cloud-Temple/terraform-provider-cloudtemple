package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSHostId   = "COMPUTE_IAAS_OPENSOURCE_HOST_ID"
	OpenIaaSHostName = "COMPUTE_IAAS_OPENSOURCE_HOST_NAME"
)

func TestAccDataSourceOpenIaaSHost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSHost, os.Getenv(OpenIaaSHostId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_host.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_host.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_host.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_host.foo", "machine_manager_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSHostName, os.Getenv(OpenIaaSHostName), os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_host.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_host.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_host.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_host.foo", "machine_manager_id"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSHostMissing,
				ExpectError: regexp.MustCompile("failed to find host with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSHost = `
data "cloudtemple_compute_iaas_opensource_host" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSHostName = `
data "cloudtemple_compute_iaas_opensource_host" "foo" {
  name               = "%s"
  machine_manager_id = "%s"
}
`

const testAccDataSourceOpenIaaSHostMissing = `
data "cloudtemple_compute_iaas_opensource_host" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
