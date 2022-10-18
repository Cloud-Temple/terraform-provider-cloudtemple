package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceCompany(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceCompany,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_company.foo", "id", "77a7d0a7-768d-4688-8c32-5fc539c5a859"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_company.foo", "name", "Cloud Temple"),
				),
			},
		},
	})
}

const testAccDataSourceCompany = `
data "cloudtemple_iam_company" "foo" {}
`
