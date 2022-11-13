package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourcePersonalAccessToken(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePersonalAccessToken,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_iam_personal_access_token.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttrSet("cloudtemple_iam_personal_access_token.foo", "client_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_iam_personal_access_token.foo", "secret_id"),
				),
			},
		},
	})
}

const testAccResourcePersonalAccessToken = `
resource "cloudtemple_iam_personal_access_token" "foo" {
  name            = "test-terraform"
  expiration_date = "2023-01-02T15:04:05Z"

  roles = [
	"c83a22e9-70bb-485e-a463-78a99484e5bb"
  ]
}
`
