package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceUsers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceUsers,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.cloudtemple_iam_users.foo", "users.*", map[string]string{
						"id":             os.Getenv(UserId),
						"internal_id":    os.Getenv(UserInternalId),
						"name":           os.Getenv(UserName),
						"type":           os.Getenv(UserType),
						"email_verified": os.Getenv(UserEmailVerified),
						"email":          os.Getenv(UserEmail),
					}),
				),
			},
		},
	})
}

const testAccDataSourceUsers = `
data "cloudtemple_iam_users" "foo" {}
`
