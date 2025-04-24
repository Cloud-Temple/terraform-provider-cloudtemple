package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	TenantId   = "TENANT_ID"
	TenantName = "TENANT_NAME"
)

func TestAccDataSourceTenants(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceTenants,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des tenants n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_iam_tenants.foo",
						"tenants.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse tenants count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected tenants list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_tenants.foo", "tenants.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_tenants.foo", "tenants.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_tenants.foo", "tenants.0.snc"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_tenants.foo", "tenants.0.company_id"),
				),
			},
		},
	})
}

const testAccDataSourceTenants = `
data "cloudtemple_iam_tenants" "foo" {}
`
