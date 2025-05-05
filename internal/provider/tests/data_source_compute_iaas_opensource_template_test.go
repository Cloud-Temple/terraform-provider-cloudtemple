package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSTemplateId   = "COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"
	OpenIaaSTemplateName = "COMPUTE_IAAS_OPENSOURCE_TEMPLATE_NAME"
)

func TestAccDataSourceOpenIaaSTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSTemplate, os.Getenv(OpenIaaSTemplateId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_template.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_template.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_template.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_template.foo", "machine_manager_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSTemplateName, os.Getenv(OpenIaaSTemplateName), os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_template.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_template.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_template.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_template.foo", "machine_manager_id"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSTemplateMissing,
				ExpectError: regexp.MustCompile("failed to find template with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSTemplate = `
data "cloudtemple_compute_iaas_opensource_template" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSTemplateName = `
data "cloudtemple_compute_iaas_opensource_template" "foo" {
  name               = "%s"
  machine_manager_id = "%s"
}
`

const testAccDataSourceOpenIaaSTemplateMissing = `
data "cloudtemple_compute_iaas_opensource_template" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
