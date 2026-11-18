package provider

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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

// TestAccResourceObjectStorageBucket_CustomWithWhitelist checks the whitelist
// the platform applies after create and after update. `whitelist` is not read
// back into the state, so the check reads it from the API.
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
					testAccCheckObjectStorageBucketPlatformWhitelist(t, "cloudtemple_object_storage_bucket.test", []string{"10.0.0.0/8", "192.168.1.0/24"}),
				),
			},
			{
				Config: testAccResourceObjectStorageBucketCustomUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_object_storage_bucket.test", "access_type", "custom"),
					testAccCheckObjectStorageBucketPlatformWhitelist(t, "cloudtemple_object_storage_bucket.test", []string{"10.0.0.0/8", "172.16.0.0/12"}),
				),
			},
		},
	})
}

// testAccCheckObjectStorageBucketPlatformWhitelist requires the whitelist the
// platform applies to the bucket to be exactly `want`, in any order.
func testAccCheckObjectStorageBucketPlatformWhitelist(t *testing.T, resourceName string, want []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		name := s.RootModule().Resources[resourceName].Primary.Attributes["name"]
		got, err := testAccClient(t).ObjectStorage().Bucket().ReadWhitelist(context.Background(), name)
		if err != nil {
			return fmt.Errorf("read the whitelist of bucket %s: %w", name, err)
		}
		got = append([]string(nil), got...)
		want = append([]string(nil), want...)
		sort.Strings(got)
		sort.Strings(want)
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("platform whitelist of bucket %s is %v, want %v", name, got, want)
		}
		return nil
	}
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

const testAccResourceObjectStorageBucketCustomUpdated = `
resource "cloudtemple_object_storage_bucket" "test" {
  name        = "terraform-test-bucket-custom"
  access_type = "custom"
  whitelist   = ["10.0.0.0/8", "172.16.0.0/12"]
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
