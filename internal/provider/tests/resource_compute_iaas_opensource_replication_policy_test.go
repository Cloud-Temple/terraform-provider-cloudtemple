package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceOpenIaaSReplicationPolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceOpenIaaSReplicationPolicy,
					os.Getenv(OpenIaaSStorageRepositoryName),
					os.Getenv(OpenIaaSMachineManagerId),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_replication_policy.foo", "id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_replication_policy.foo", "name", "test-terraform-replication-policy"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_replication_policy.foo", "storage_repository_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_replication_policy.foo", "interval.0.hours", "6"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_replication_policy.foo", "pool_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_replication_policy.foo", "machine_manager_id"),
				),
			},
		},
	})
}

const testAccResourceOpenIaaSReplicationPolicy = `
data "cloudtemple_compute_iaas_opensource_storage_repository" "sr" {
  name               = "%s"
  machine_manager_id = "%s"
}

resource "cloudtemple_compute_iaas_opensource_replication_policy" "foo" {
  name                  = "test-terraform-replication-policy"
  storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repository.sr.id

  interval {
    hours = 6
  }
}
`
