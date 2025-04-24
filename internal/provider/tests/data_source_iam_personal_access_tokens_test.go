package provider

import (
	"fmt"
	"strconv"
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
					// Vérifier que la liste des tokens n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_iam_personal_access_tokens.foo",
						"tokens.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse tokens count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected tokens list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_personal_access_tokens.foo", "tokens.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_personal_access_tokens.foo", "tokens.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_personal_access_tokens.foo", "tokens.0.expiration_date"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_personal_access_tokens.foo", "tokens.0.roles.#"),
				),
			},
		},
	})
}

const testAccDataSourcePersonalAccessTokens = `
data "cloudtemple_iam_personal_access_tokens" "foo" {}
`
