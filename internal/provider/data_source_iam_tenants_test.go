package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceTenants(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceTenants,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_tenants.foo", "tenants.#", "1"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_tenants.foo", "tenants.0.id"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_tenants.foo", "tenants.0.name", "BOB"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_tenants.foo", "tenants.0.snc", "false"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_tenants.foo", "tenants.0.company_id"),
				),
			},
		},
	})
}

const testAccDataSourceTenants = `
data "cloudtemple_iam_tenants" "foo" {}
`
