package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRoles(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRoles,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_roles.foo", "roles.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_roles.foo", "roles.0.name"),
				),
			},
		},
	})
}

const testAccDataSourceRoles = `
data "cloudtemple_iam_roles" "foo" {}
`
