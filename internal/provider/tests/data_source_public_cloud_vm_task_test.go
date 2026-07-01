package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMTaskId = "PUBLIC_CLOUD_VM_TASK_ID"
)

func TestAccDataSourcePublicCloudVMTask(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMTask, os.Getenv(PublicCloudVMTaskId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_task.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_task.foo", "status"),
				),
			},
			{
				Config:      testAccDataSourcePublicCloudVMTaskMissing,
				ExpectError: regexp.MustCompile("failed to find task with id"),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMTask = `
data "cloudtemple_public_cloud_vm_task" "foo" {
  id = "%s"
}
`

const testAccDataSourcePublicCloudVMTaskMissing = `
data "cloudtemple_public_cloud_vm_task" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
