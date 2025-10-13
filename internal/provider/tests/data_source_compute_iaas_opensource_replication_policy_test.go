package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	ReplicationPolicyId   = "COMPUTE_IAAS_OPENSOURCE_REPLICATION_POLICY_ID"
	ReplicationPolicyName = "COMPUTE_IAAS_OPENSOURCE_REPLICATION_POLICY_NAME"
)

func TestAccDataSourceOpenIaaSReplicationPolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSReplicationPolicy, os.Getenv(ReplicationPolicyId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_iaas_opensource_replication_policy.foo", "id", os.Getenv(ReplicationPolicyId)),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_replication_policy.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_replication_policy.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_replication_policy.foo", "storage_repository_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_replication_policy.foo", "pool_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_replication_policy.foo", "interval.#"),
				),
			},
		},
	})
}

const testAccDataSourceOpenIaaSReplicationPolicy = `
data "cloudtemple_compute_iaas_opensource_replication_policy" "foo" {
  id = "%s"
}
`
