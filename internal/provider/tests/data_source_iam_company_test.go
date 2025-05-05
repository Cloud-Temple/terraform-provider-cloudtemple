package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceCompany(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceCompany, os.Getenv(testCompanyIDEnvName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_company.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_company.foo", "name"),
				),
			},
			{
				Config:      testAccDataSourceCompanyMissing,
				ExpectError: regexp.MustCompile("failed to find company with id"),
			},
		},
	})
}

const testAccDataSourceCompany = `
data "cloudtemple_iam_company" "foo" {
  id = "%s"
}
`

const testAccDataSourceCompanyMissing = `
data "cloudtemple_iam_company" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
