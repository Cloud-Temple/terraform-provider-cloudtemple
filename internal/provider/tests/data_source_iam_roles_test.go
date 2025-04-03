package provider

import (
	"os"
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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("data.cloudtemple_iam_roles.foo", "roles.*", map[string]string{
						"id":   os.Getenv(RoleId),
						"name": os.Getenv(RoleName),
					}),
				),
			},
		},
	})
}

const testAccDataSourceRoles = `
data "cloudtemple_iam_roles" "foo" {}
`
