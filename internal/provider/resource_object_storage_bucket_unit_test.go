package provider

import (
	"reflect"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestBuildBucketIPWhitelist pins the mapping from the `whitelist` attribute to
// the `ipWhiteList` entries sent on create and update: every CIDR, in order,
// duplicates kept, each entry carrying only its CIDR. This list is the bucket's
// whole access boundary, and the acceptance test that also covers it is not run
// in CI.
func TestBuildBucketIPWhitelist(t *testing.T) {
	t.Run("maps every CIDR, in order, to an entry carrying only that CIDR", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, resourceBucket().Schema, map[string]interface{}{
			"name":        "bucket-1",
			"access_type": "custom",
			"whitelist":   []interface{}{"10.0.0.0/8", "192.168.1.0/24", "2001:db8::/48", "10.0.0.0/8"},
		})
		got := buildBucketIPWhitelist(d)
		want := []client.IPWhitelistEntry{
			{CIDR: "10.0.0.0/8"},
			{CIDR: "192.168.1.0/24"},
			{CIDR: "2001:db8::/48"},
			{CIDR: "10.0.0.0/8"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected entries:\n got  %+v\n want %+v", got, want)
		}
	})

	// nil and an empty slice serialize the same under `omitempty`: no list is
	// sent, which is what a private or public bucket needs.
	t.Run("no whitelist gives no entry", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, resourceBucket().Schema, map[string]interface{}{
			"name":        "bucket-1",
			"access_type": "private",
		})
		if got := buildBucketIPWhitelist(d); len(got) != 0 {
			t.Fatalf("expected no entry, got %+v", got)
		}
	})
}
