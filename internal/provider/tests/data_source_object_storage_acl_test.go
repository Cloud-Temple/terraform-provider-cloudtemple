package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceObjectStorageACL_ByBucket(t *testing.T) {
	bucketName := os.Getenv("OBJECT_STORAGE_BUCKET_NAME")
	if bucketName == "" {
		t.Skip("OBJECT_STORAGE_BUCKET_NAME not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceObjectStorageACLByBucket, bucketName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_acl.foo", "acls.#"),
				),
			},
		},
	})
}

func TestAccDataSourceObjectStorageACL_ByStorageAccount(t *testing.T) {
	storageAccountName := os.Getenv("OBJECT_STORAGE_STORAGE_ACCOUNT_NAME")
	if storageAccountName == "" {
		t.Skip("OBJECT_STORAGE_STORAGE_ACCOUNT_NAME not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceObjectStorageACLByStorageAccount, storageAccountName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_acl.foo", "acls.#"),
				),
			},
		},
	})
}

func TestAccDataSourceObjectStorageACL_NeitherParameter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceObjectStorageACLNeitherParameter,
				ExpectError: regexp.MustCompile("either bucket_name or storage_account_name must be specified"),
			},
		},
	})
}

const testAccDataSourceObjectStorageACLByBucket = `
data "cloudtemple_object_storage_acl" "foo" {
  bucket_name = "%s"
}
`

const testAccDataSourceObjectStorageACLByStorageAccount = `
data "cloudtemple_object_storage_acl" "foo" {
  storage_account_name = "%s"
}
`

const testAccDataSourceObjectStorageACLNeitherParameter = `
data "cloudtemple_object_storage_acl" "foo" {}
`
