package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourcePersonalAccessToken(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourcePersonalAccessToken, os.Getenv(PatName), os.Getenv(RoleId)),
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
  name            = "%s"
  expiration_date = "2023-11-02T15:04:05Z"

  roles = [
	"%s"
  ]
}
`
