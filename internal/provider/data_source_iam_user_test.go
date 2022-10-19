package provider

import (
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
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "id", "37105598-4889-43da-82ea-cf60f2a36aee"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "internal_id", "7b8ba092-52e3-4c21-a2f5-adca40a80d34"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "name", "Rémi Lapeyre"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "type", "LocalAccount"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "source.#", "0"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "email_verified", "true"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_user.foo", "email", "remi.lapeyre@lenstra.fr"),
				),
			},
		},
	})
}

const testAccDataSourceUser = `
data "cloudtemple_iam_user" "foo" {
  id = "37105598-4889-43da-82ea-cf60f2a36aee"
}
`
