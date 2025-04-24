package provider

import (
	"fmt"
	"strconv"
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
					// Vérifier que la liste des users n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_iam_users.foo",
						"users.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse users count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected users list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.email_verified"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_users.foo", "users.0.email"),
				),
			},
		},
	})
}

const testAccDataSourceUsers = `
data "cloudtemple_iam_users" "foo" {}
`
