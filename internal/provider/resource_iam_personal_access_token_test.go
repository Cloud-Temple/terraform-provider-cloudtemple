package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourcePersonalAccessToken(t *testing.T) {
	// current date + 1 day
	t1 := time.Now().AddDate(0, 0, 1)
	expirationDate := t1.Format(time.RFC3339)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourcePersonalAccessToken, expirationDate, os.Getenv(RoleId)),
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
  expiration_date = "%s"

  roles = [
	"%s"
  ]
}
`
