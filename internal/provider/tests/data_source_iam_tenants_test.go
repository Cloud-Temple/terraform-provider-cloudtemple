package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	TenantId   = "TENANT_ID"
	TenantName = "TENANT_NAME"
	TenantSNC  = "TENANT_SNC"
	TenantQty  = "TENANT_QTY"
)

func TestAccDataSourceTenants(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceTenants,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_tenants.foo", "tenants.#", os.Getenv(TenantQty)),
					resource.TestCheckTypeSetElemNestedAttrs("data.cloudtemple_iam_tenants.foo", "tenants.*", map[string]string{
						"id":         os.Getenv(TenantId),
						"name":       os.Getenv(TenantName),
						"snc":        os.Getenv(TenantSNC),
						"company_id": os.Getenv(testCompanyIDEnvName),
					},
					),
				),
			},
		},
	})
}

const testAccDataSourceTenants = `
data "cloudtemple_iam_tenants" "foo" {}
`
