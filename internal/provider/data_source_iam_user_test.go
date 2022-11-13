package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceUser(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceUser,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "id", "37105598-4889-43da-82ea-cf60f2a36aee"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "internal_id", "7b8ba092-52e3-4c21-a2f5-adca40a80d34"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "name", "Rémi Lapeyre"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "type", "LocalAccount"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "source.#", "0"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "email_verified", "true"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "email", "remi.lapeyre@lenstra.fr"),
				),
			},
			{
				Config: testAccDataSourceUserName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "id", "37105598-4889-43da-82ea-cf60f2a36aee"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "internal_id", "7b8ba092-52e3-4c21-a2f5-adca40a80d34"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "name", "Rémi Lapeyre"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "type", "LocalAccount"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "source.#", "0"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "email_verified", "true"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "email", "remi.lapeyre@lenstra.fr"),
				),
			},
			{
				Config: testAccDataSourceUserInternalId,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "id", "37105598-4889-43da-82ea-cf60f2a36aee"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "internal_id", "7b8ba092-52e3-4c21-a2f5-adca40a80d34"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "name", "Rémi Lapeyre"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "type", "LocalAccount"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "source.#", "0"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "email_verified", "true"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "email", "remi.lapeyre@lenstra.fr"),
				),
			},
			{
				Config: testAccDataSourceUserEmail,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "id", "37105598-4889-43da-82ea-cf60f2a36aee"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "internal_id", "7b8ba092-52e3-4c21-a2f5-adca40a80d34"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "name", "Rémi Lapeyre"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "type", "LocalAccount"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "source.#", "0"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "email_verified", "true"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "email", "remi.lapeyre@lenstra.fr"),
				),
			},
			{
				Config:      testAccDataSourceUserMissing,
				ExpectError: regexp.MustCompile("failed to find user with id"),
			},
		},
	})
}

const testAccDataSourceUser = `
data "cloudtemple_iam_user" "foo" {
  id = "37105598-4889-43da-82ea-cf60f2a36aee"
}
`

const testAccDataSourceUserName = `
data "cloudtemple_iam_user" "foo" {
  name = "Rémi Lapeyre"
}
`

const testAccDataSourceUserInternalId = `
data "cloudtemple_iam_user" "foo" {
  internal_id = "7b8ba092-52e3-4c21-a2f5-adca40a80d34"
}
`

const testAccDataSourceUserEmail = `
data "cloudtemple_iam_user" "foo" {
  email = "remi.lapeyre@lenstra.fr"
}
`

const testAccDataSourceUserMissing = `
data "cloudtemple_iam_user" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
