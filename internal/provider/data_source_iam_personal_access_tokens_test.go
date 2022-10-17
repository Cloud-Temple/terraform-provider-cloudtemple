package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePersonalAccessTokens(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePersonalAccessTokens,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_tokens.foo", "tokens.#", "1"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_personal_access_tokens.foo", "tokens.0.id"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_tokens.foo", "tokens.0.name", "Terraform"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_tokens.foo", "tokens.0.expiration_date", "1669762800000"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_personal_access_tokens.foo", "tokens.0.roles.#"),
				),
			},
		},
	})
}

const testAccDataSourcePersonalAccessTokens = `
data "cloudtemple_iam_personal_access_tokens" "foo" {}
`
