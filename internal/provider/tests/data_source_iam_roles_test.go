package provider

import (
	"fmt"
	"strconv"
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
					// Vérifier que la liste des roles n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_iam_roles.foo",
						"roles.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse roles count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected roles list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_roles.foo", "roles.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_roles.foo", "roles.0.name"),
				),
			},
		},
	})
}

const testAccDataSourceRoles = `
data "cloudtemple_iam_roles" "foo" {}
`
