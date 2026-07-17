package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMTemplateId   = "PUBLIC_CLOUD_VM_TEMPLATE_ID"
	PublicCloudVMTemplateName = "PUBLIC_CLOUD_VM_TEMPLATE_NAME"
)

func TestAccDataSourcePublicCloudVMTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMTemplate, os.Getenv(PublicCloudVMTemplateId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_template.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_template.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_template.foo", "os_family"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMTemplateName, os.Getenv(PublicCloudVMTemplateName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_template.foo", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_template.foo", "name", os.Getenv(PublicCloudVMTemplateName)),
				),
			},
			{
				Config:      fmt.Sprintf(testAccDataSourcePublicCloudVMTemplateConflict, os.Getenv(PublicCloudVMTemplateId), os.Getenv(PublicCloudVMTemplateName)),
				ExpectError: regexp.MustCompile(`"id": conflicts with name`),
			},
			{
				Config:      testAccDataSourcePublicCloudVMTemplateMissing,
				ExpectError: regexp.MustCompile("failed to find template with id"),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMTemplate = `
data "cloudtemple_public_cloud_vm_template" "foo" {
  id = "%s"
}
`

const testAccDataSourcePublicCloudVMTemplateName = `
data "cloudtemple_public_cloud_vm_template" "foo" {
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMTemplateConflict = `
data "cloudtemple_public_cloud_vm_template" "foo" {
  id   = "%s"
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMTemplateMissing = `
data "cloudtemple_public_cloud_vm_template" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
