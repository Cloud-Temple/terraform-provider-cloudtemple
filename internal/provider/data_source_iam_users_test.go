package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceUsers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceUsers,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.source.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.email_verified"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.email"),
				),
			},
		},
	})
}

const testAccDataSourceUsers = `
data "cloudtemple_iam_users" "foo" {}
`
