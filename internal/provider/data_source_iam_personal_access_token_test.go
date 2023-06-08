package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PatName     = "IAM_PAT_NAME"
	PatRolesQty = "IAM_PAT_ROLE_QTY"
)

func TestAccDataSourcePersonalAccessToken(t *testing.T) {

	expirationDate := time.Now().AddDate(0, 0, 1)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePersonalAccessToken, expirationDate.Format(time.RFC3339), os.Getenv(RoleId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "roles.0", os.Getenv(RoleId)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourcePersonalAccessTokenName, expirationDate.Format(time.RFC3339), os.Getenv(RoleId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_personal_access_token.foo", "roles.0", os.Getenv(RoleId)),
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
  expiration_date = "%s"

  roles = [
    "%s"
  ]
}

data "cloudtemple_iam_personal_access_token" "foo" {
  id = cloudtemple_iam_personal_access_token.foo.id
}
`

const testAccDataSourcePersonalAccessTokenName = `
resource "cloudtemple_iam_personal_access_token" "foo" {
  name            = "test-terraform"
  expiration_date = "%s"

  roles = [
    "%s"
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
