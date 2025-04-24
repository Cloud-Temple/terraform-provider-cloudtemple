package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	UserEmail         = "IAM_USER_EMAIL"
	UserEmailVerified = "IAM_USER_EMAIL_VERIFIED"
	UserId            = "IAM_USER_ID"
	UserInternalId    = "IAM_USER_INTERNAL_ID"
	UserName          = "IAM_USER_NAME"
	UserType          = "IAM_USER_TYPE"
)

func TestAccDataSourceUser(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceUser, os.Getenv(UserId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "email"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "email_verified"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceUserName, os.Getenv(UserName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "email"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "email_verified"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceUserInternalId, os.Getenv(UserInternalId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "email"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "email_verified"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceUserEmail, os.Getenv(UserEmail)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "email"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_user.foo", "email_verified"),
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
  id = "%s"
}
`

const testAccDataSourceUserName = `
data "cloudtemple_iam_user" "foo" {
  name = "%s"
}
`

const testAccDataSourceUserInternalId = `
data "cloudtemple_iam_user" "foo" {
  internal_id = "%s"
}
`

const testAccDataSourceUserEmail = `
data "cloudtemple_iam_user" "foo" {
  email = "%s"
}
`

const testAccDataSourceUserMissing = `
data "cloudtemple_iam_user" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
