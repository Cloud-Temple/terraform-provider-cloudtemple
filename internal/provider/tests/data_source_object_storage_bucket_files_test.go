package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceObjectStorageBucketFiles(t *testing.T) {
	bucketName := os.Getenv("OBJECT_STORAGE_BUCKET_NAME")
	if bucketName == "" {
		t.Skip("OBJECT_STORAGE_BUCKET_NAME not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceObjectStorageBucketFiles, bucketName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_bucket_files.foo", "files.#"),
				),
			},
		},
	})
}

func TestAccDataSourceObjectStorageBucketFiles_WithFolderPath(t *testing.T) {
	bucketName := os.Getenv("OBJECT_STORAGE_BUCKET_NAME")
	if bucketName == "" {
		t.Skip("OBJECT_STORAGE_BUCKET_NAME not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceObjectStorageBucketFilesWithFolderPath, bucketName, "subfolder"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_bucket_files.foo", "files.#"),
				),
			},
		},
	})
}

const testAccDataSourceObjectStorageBucketFiles = `
data "cloudtemple_object_storage_bucket_files" "foo" {
  bucket_name = "%s"
}
`

const testAccDataSourceObjectStorageBucketFilesWithFolderPath = `
data "cloudtemple_object_storage_bucket_files" "foo" {
  bucket_name = "%s"
  folder_path = "%s"
}
`
