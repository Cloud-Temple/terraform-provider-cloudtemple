package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRole(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRole,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_role.foo", "id", "c83a22e9-70bb-485e-a463-78a99484e5bb"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_role.foo", "name", "compute_read"),
				),
			},
			{
				Config:      testAccDataSourceRoleMissing,
				ExpectError: regexp.MustCompile("failed to find role with id"),
			},
		},
	})
}

const testAccDataSourceRole = `
data "cloudtemple_iam_role" "foo" {
  id = "c83a22e9-70bb-485e-a463-78a99484e5bb"
}
`

const testAccDataSourceRoleMissing = `
data "cloudtemple_iam_role" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
