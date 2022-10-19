package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePersonalAccessToken(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePersonalAccessToken,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "roles.0", "c83a22e9-70bb-485e-a463-78a99484e5bb"),
				),
			},
		},
	})
}

const testAccDataSourcePersonalAccessToken = `
resource "cloudtemple_iam_personal_access_token" "foo" {
  name            = "test-terraform"
  expiration_date = "2023-01-02T15:04:05Z"
  
  roles = [
    "c83a22e9-70bb-485e-a463-78a99484e5bb"
  ]
}

data "cloudtemple_iam_personal_access_token" "foo" {
  client_id = cloudtemple_iam_personal_access_token.foo.id
}
`
