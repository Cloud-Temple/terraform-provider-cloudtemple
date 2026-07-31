package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMImageId   = "PUBLIC_CLOUD_VM_IMAGE_ID"
	PublicCloudVMImageName = "PUBLIC_CLOUD_VM_IMAGE_NAME"
)

// skipIfNoPublicCloudVMImageEnv skips a live image test unless the image
// identifiers are provided. The GET /vm_instances/v1/images endpoint is only
// deployed on DEV (the PROD broker switch is still pending), while the
// provider's default API address is PROD — so running these tests against PROD
// would hit a missing endpoint. Requiring the operator to declare a DEV image
// id/name is the explicit signal that the target environment actually serves
// /images; without it, the test skips instead of failing against PROD.
func skipIfNoPublicCloudVMImageEnv(t *testing.T) {
	if os.Getenv(PublicCloudVMImageId) == "" || os.Getenv(PublicCloudVMImageName) == "" {
		t.Skip(PublicCloudVMImageId + " / " + PublicCloudVMImageName + " not set (the /vm_instances/v1/images endpoint is DEV-only; set them to a DEV image to run this test)")
	}
}

func TestAccDataSourcePublicCloudVMImage(t *testing.T) {
	skipIfNoPublicCloudVMImageEnv(t)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMImage, os.Getenv(PublicCloudVMImageId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_image.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_image.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_image.foo", "os_family"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMImageName, os.Getenv(PublicCloudVMImageName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_image.foo", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_image.foo", "name", os.Getenv(PublicCloudVMImageName)),
				),
			},
			{
				Config:      fmt.Sprintf(testAccDataSourcePublicCloudVMImageConflict, os.Getenv(PublicCloudVMImageId), os.Getenv(PublicCloudVMImageName)),
				ExpectError: regexp.MustCompile(`"id": conflicts with name`),
			},
			{
				Config:      testAccDataSourcePublicCloudVMImageMissing,
				ExpectError: regexp.MustCompile("failed to find image with id"),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMImage = `
data "cloudtemple_public_cloud_vm_image" "foo" {
  id = "%s"
}
`

const testAccDataSourcePublicCloudVMImageName = `
data "cloudtemple_public_cloud_vm_image" "foo" {
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMImageConflict = `
data "cloudtemple_public_cloud_vm_image" "foo" {
  id   = "%s"
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMImageMissing = `
data "cloudtemple_public_cloud_vm_image" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
