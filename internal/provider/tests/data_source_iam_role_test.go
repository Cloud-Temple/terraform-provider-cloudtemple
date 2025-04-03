package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	RoleId   = "IAM_ROLE_ID"
	RoleName = "IAM_ROLE_NAME"
)

func TestAccDataSourceRole(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceRole, os.Getenv(RoleId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_role.foo", "id", os.Getenv(RoleId)),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_role.foo", "name", os.Getenv(RoleName)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceRoleName, os.Getenv(RoleName)),
			},
			{
				Config:      fmt.Sprintf(testAccDataSourceRoleConflict, os.Getenv(RoleId), os.Getenv(RoleName)),
				ExpectError: regexp.MustCompile(`"id": conflicts with name`),
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
  id = "%s"
}
`

const testAccDataSourceRoleName = `
data "cloudtemple_iam_role" "foo" {
  name = "%s"
}
`

const testAccDataSourceRoleConflict = `
data "cloudtemple_iam_role" "foo" {
  id   = "%s"
  name = "%s"
}
`

const testAccDataSourceRoleMissing = `
data "cloudtemple_iam_role" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
