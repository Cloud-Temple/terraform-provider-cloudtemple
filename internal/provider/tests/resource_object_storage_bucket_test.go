package provider

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
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

// TestAccResourceObjectStorageBucket_CustomWithWhitelist proves, against the
// real API, that the whitelist sent as `ipWhiteList` is the one the platform
// records — on create (POST /buckets) and on update (PUT /buckets/{b}/whitelist).
// The `whitelist` attribute is write-only (never read back into the state), so
// a state assertion would only echo the configuration: each step reads the
// whitelist from the platform instead, and the update step keeps one range,
// drops one and adds one, so that the whole list is proven replaced.
func TestAccResourceObjectStorageBucket_CustomWithWhitelist(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckObjectStorageBucketDestroyed(t, "terraform-test-bucket-custom"),
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

// testAccCheckObjectStorageBucketPlatformWhitelist reads the bucket whitelist
// from the API (not from the state) and requires it to be exactly `want`: same
// ranges, no extra one, no missing one, no duplicate. The platform returns the
// entries in its own order, so both sides are compared sorted.
func testAccCheckObjectStorageBucketPlatformWhitelist(t *testing.T, resourceName string, want []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s not found in state", resourceName)
		}
		name := rs.Primary.Attributes["name"]
		if name == "" {
			return fmt.Errorf("%s has no name in state", resourceName)
		}

		entries, err := testAccClient(t).ObjectStorage().Bucket().ReadWhitelist(context.Background(), name)
		if err != nil {
			return fmt.Errorf("read the whitelist of bucket %s from the platform: %w", name, err)
		}
		got := make([]string, 0, len(entries))
		for _, e := range entries {
			got = append(got, e.CIDR)
		}
		t.Logf("platform whitelist of bucket %s: %+v", name, entriesForLog(entries))

		wantSorted := append([]string(nil), want...)
		sort.Strings(got)
		sort.Strings(wantSorted)
		if !reflect.DeepEqual(got, wantSorted) {
			return fmt.Errorf("platform whitelist of bucket %s is %v, want %v", name, got, wantSorted)
		}
		return nil
	}
}

func entriesForLog(entries []*client.BucketWhitelistEntry) []client.BucketWhitelistEntry {
	out := make([]client.BucketWhitelistEntry, len(entries))
	for i, e := range entries {
		out[i] = *e
	}
	return out
}

// testAccCheckObjectStorageBucketDestroyed requires the bucket to be absent
// from a complete listing once the test has destroyed it. A complete HTTP 200
// listing is the only acceptable proof of absence (#281), and a JSON `null`
// body is not a listing: it decodes to a nil slice that would otherwise read
// as "no bucket at all".
func testAccCheckObjectStorageBucketDestroyed(t *testing.T, name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		buckets, err := testAccClient(t).ObjectStorage().Bucket().ListStrict(context.Background())
		if err != nil {
			return fmt.Errorf("list the buckets to confirm the deletion of %s: %w", name, err)
		}
		if buckets == nil {
			return fmt.Errorf("the bucket listing is null, so it cannot prove that %s was deleted", name)
		}
		for _, b := range buckets {
			if b.Name == name {
				return fmt.Errorf("bucket %s still exists after destroy", name)
			}
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

// Keeps 10.0.0.0/8, drops 192.168.1.0/24, adds 172.16.0.0/12.
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
