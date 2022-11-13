package provider

import (
	"regexp"
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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "roles.0", "c83a22e9-70bb-485e-a463-78a99484e5bb"),
				),
			},
			{
				Config: testAccDataSourcePersonalAccessTokenName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "roles.0", "c83a22e9-70bb-485e-a463-78a99484e5bb"),
				),
			},
			{
				Config:      testAccDataSourcePersonalAccessTokenMissing,
				ExpectError: regexp.MustCompile("failed to find personal access token with id"),
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
  id = cloudtemple_iam_personal_access_token.foo.id
}
`

const testAccDataSourcePersonalAccessTokenName = `
resource "cloudtemple_iam_personal_access_token" "foo" {
  name            = "test-terraform"
  expiration_date = "2023-01-02T15:04:05Z"

  roles = [
    "c83a22e9-70bb-485e-a463-78a99484e5bb"
  ]
}

data "cloudtemple_iam_personal_access_token" "foo" {
  name = "test-terraform"
}
`

const testAccDataSourcePersonalAccessTokenMissing = `
data "cloudtemple_iam_personal_access_token" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
