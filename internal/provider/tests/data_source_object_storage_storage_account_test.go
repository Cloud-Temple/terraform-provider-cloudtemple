package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	testStorageAccountNameEnvName = "OBJECT_STORAGE_STORAGE_ACCOUNT_NAME"
)

func TestAccDataSourceObjectStorageStorageAccount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckObjectStorageStorageAccount(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceObjectStorageStorageAccount, os.Getenv(testStorageAccountNameEnvName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_storage_account.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_storage_account.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_storage_account.foo", "access_key_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_storage_account.foo", "arn"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_storage_account.foo", "path"),
				),
			},
			{
				Config:      testAccDataSourceObjectStorageStorageAccountMissing,
				ExpectError: regexp.MustCompile("error reading storage account with name"),
			},
		},
	})
}

func testAccPreCheckObjectStorageStorageAccount(t *testing.T) {
	if v := os.Getenv(testStorageAccountNameEnvName); v == "" {
		t.Fatalf("%s must be set for acceptance tests", testStorageAccountNameEnvName)
	}
}

const testAccDataSourceObjectStorageStorageAccount = `
data "cloudtemple_object_storage_storage_account" "foo" {
  name = "%s"
}
`

const testAccDataSourceObjectStorageStorageAccountMissing = `
data "cloudtemple_object_storage_storage_account" "foo" {
  name = "non-existent-storage-account-terraform-test-12345"
}
`
