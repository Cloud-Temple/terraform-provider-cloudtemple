package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	HostClusterId     = "COMPUTE_HOST_CLUSTER_ID"
	HostClusterName   = "COMPUTE_HOST_CLUSTER_NAME"
	MachineManagerId2 = "COMPUTE_VCENTER_ID_2"
)

func TestAccDataSourceHostCluster(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceHostCluster, os.Getenv(HostClusterId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "hosts.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "metrics.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "virtual_machines_number"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceHostClusterName, os.Getenv(HostClusterName), os.Getenv(MachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "hosts.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "metrics.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_cluster.foo", "virtual_machines_number"),
				),
			},
			{
				Config:      testAccDataSourceHostClusterMissing,
				ExpectError: regexp.MustCompile("failed to find host cluster with id"),
			},
		},
	})
}

const testAccDataSourceHostCluster = `
data "cloudtemple_compute_host_cluster" "foo" {
  id = "%s"
}
`

const testAccDataSourceHostClusterName = `
data "cloudtemple_compute_host_cluster" "foo" {
  name               = "%s"
	machine_manager_id = "%s"
}
`

const testAccDataSourceHostClusterMissing = `
data "cloudtemple_compute_host_cluster" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
