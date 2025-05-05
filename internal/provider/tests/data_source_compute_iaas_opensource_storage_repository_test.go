package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSStorageRepositoryId   = "COMPUTE_IAAS_OPENSOURCE_STORAGE_REPOSITORY_ID"
	OpenIaaSStorageRepositoryName = "COMPUTE_IAAS_OPENSOURCE_STORAGE_REPOSITORY_NAME"
)

func TestAccDataSourceOpenIaaSStorageRepository(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSStorageRepository, os.Getenv(OpenIaaSStorageRepositoryId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repository.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repository.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repository.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repository.foo", "machine_manager_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSStorageRepositoryName, os.Getenv(OpenIaaSStorageRepositoryName), os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repository.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repository.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repository.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repository.foo", "machine_manager_id"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSStorageRepositoryMissing,
				ExpectError: regexp.MustCompile("failed to find storage repository with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSStorageRepository = `
data "cloudtemple_compute_iaas_opensource_storage_repository" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSStorageRepositoryName = `
data "cloudtemple_compute_iaas_opensource_storage_repository" "foo" {
  name               = "%s"
  machine_manager_id = "%s"
}
`

const testAccDataSourceOpenIaaSStorageRepositoryMissing = `
data "cloudtemple_compute_iaas_opensource_storage_repository" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
