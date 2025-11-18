package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	testBucketNameEnvName = "OBJECT_STORAGE_BUCKET_NAME"
)

func TestAccDataSourceObjectStorageBucket(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckObjectStorageBucket(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceObjectStorageBucket, os.Getenv(testBucketNameEnvName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_bucket.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_bucket.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_bucket.foo", "namespace"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_bucket.foo", "endpoint"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_bucket.foo", "versioning"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_bucket.foo", "total_objects"),
				),
			},
			{
				Config:      testAccDataSourceObjectStorageBucketMissing,
				ExpectError: regexp.MustCompile("error reading bucket with name"),
			},
		},
	})
}

func testAccPreCheckObjectStorageBucket(t *testing.T) {
	if v := os.Getenv(testBucketNameEnvName); v == "" {
		t.Fatalf("%s must be set for acceptance tests", testBucketNameEnvName)
	}
}

const testAccDataSourceObjectStorageBucket = `
data "cloudtemple_object_storage_bucket" "foo" {
  name = "%s"
}
`

const testAccDataSourceObjectStorageBucketMissing = `
data "cloudtemple_object_storage_bucket" "foo" {
  name = "non-existent-bucket-terraform-test-12345"
}
`
