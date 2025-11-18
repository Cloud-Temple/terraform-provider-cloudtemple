package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceObjectStorageBucket_Private(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceObjectStorageBucketPrivate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_object_storage_bucket.test", "name", "terraform-test-bucket-private"),
					resource.TestCheckResourceAttr("cloudtemple_object_storage_bucket.test", "access_type", "private"),
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_bucket.test", "id"),
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_bucket.test", "endpoint"),
				),
			},
		},
	})
}

func TestAccResourceObjectStorageBucket_CustomWithWhitelist(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceObjectStorageBucketCustom,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_object_storage_bucket.test", "name", "terraform-test-bucket-custom"),
					resource.TestCheckResourceAttr("cloudtemple_object_storage_bucket.test", "access_type", "custom"),
					resource.TestCheckResourceAttr("cloudtemple_object_storage_bucket.test", "whitelist.#", "2"),
				),
			},
		},
	})
}

func TestAccResourceObjectStorageBucket_WithVersioning(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceObjectStorageBucketVersioning,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_object_storage_bucket.test", "name", "terraform-test-bucket-versioning"),
					resource.TestCheckResourceAttr("cloudtemple_object_storage_bucket.test", "versioning", "Enabled"),
				),
			},
			{
				Config: testAccResourceObjectStorageBucketVersioningSuspended,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_object_storage_bucket.test", "versioning", "Suspended"),
				),
			},
		},
	})
}

const testAccResourceObjectStorageBucketPrivate = `
resource "cloudtemple_object_storage_bucket" "test" {
  name        = "terraform-test-bucket-private"
  access_type = "private"
}
`

const testAccResourceObjectStorageBucketCustom = `
resource "cloudtemple_object_storage_bucket" "test" {
  name        = "terraform-test-bucket-custom"
  access_type = "custom"
  whitelist   = ["10.0.0.0/8", "192.168.1.0/24"]
}
`

const testAccResourceObjectStorageBucketVersioning = `
resource "cloudtemple_object_storage_bucket" "test" {
  name        = "terraform-test-bucket-versioning"
  access_type = "private"
  versioning  = "Enabled"
}
`

const testAccResourceObjectStorageBucketVersioningSuspended = `
resource "cloudtemple_object_storage_bucket" "test" {
  name        = "terraform-test-bucket-versioning"
  access_type = "private"
  versioning  = "Suspended"
}
`
