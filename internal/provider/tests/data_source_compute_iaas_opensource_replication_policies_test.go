package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOpenIaaSReplicationPolicies(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpenIaaSReplicationPolicies,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_replication_policies.foo",
						"policies.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse policies count: %s", err)
							}
							if count < 0 {
								return fmt.Errorf("expected policies count to be >= 0, got %d", count)
							}
							return nil
						},
					),
				),
			},
		},
	})
}

const testAccDataSourceOpenIaaSReplicationPolicies = `
data "cloudtemple_compute_iaas_opensource_replication_policies" "foo" {}
`
