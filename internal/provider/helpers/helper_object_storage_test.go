package helpers

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestFlattenStorageAccountNeverEmitsTheSecretKey is the dedicated state-secret
// test for object storage accounts. The storage-account resource exposes a
// create-only `access_secret_key` (Sensitive, only returned at creation) that
// the resource Read deliberately does NOT overwrite. FlattenStorageAccount must
// therefore NEVER emit any secret-key attribute: if it did, every refresh would
// overwrite the only stored copy of the secret with an empty string (the
// #264 Lot D state-secret failure mode), silently destroying it.
//
// FlattenStorageAccount must emit `access_key_id` (the public key id, safe to
// refresh) but not the secret. This test pins both: the public id is preserved,
// and no secret key is emitted by name or value.
func TestFlattenStorageAccountNeverEmitsTheSecretKey(t *testing.T) {
	account := &client.StorageAccount{
		ID:          "acct-1",
		Name:        "backups",
		AccessKeyID: "AKIAEXAMPLE",
		ARN:         "arn:aws:s3:::backups",
		CreateDate:  "2026-01-01T00:00:00Z",
		Path:        "/team/",
	}
	account.Tags = append(account.Tags, struct {
		Key   string
		Value string
	}{Key: "env", Value: "prod"})

	got := FlattenStorageAccount(account)

	for _, forbidden := range []string{"access_secret_key", "secret", "secret_access_key", "SecretAccessKey"} {
		if _, present := got[forbidden]; present {
			t.Errorf("FlattenStorageAccount emits forbidden secret key %q; on refresh it would overwrite the create-only secret with empty and destroy it (#264 Lot D)", forbidden)
		}
	}

	// The public access key id is preserved.
	assertEq(t, "access_key_id", got["access_key_id"], "AKIAEXAMPLE")
	assertEq(t, "id", got["id"], "acct-1")
	assertEq(t, "name", got["name"], "backups")
	assertEq(t, "arn", got["arn"], "arn:aws:s3:::backups")
	assertEq(t, "create_date", got["create_date"], "2026-01-01T00:00:00Z")
	assertEq(t, "path", got["path"], "/team/")

	// Tags are a list of {key,value} blocks, preserved in order.
	tags, ok := got["tags"].([]map[string]interface{})
	if !ok {
		t.Fatalf("tags has type %T, want []map[string]interface{}", got["tags"])
	}
	if len(tags) != 1 {
		t.Fatalf("tags has %d elements, want 1", len(tags))
	}
	assertEq(t, "tags[0].key", tags[0]["key"], "env")
	assertEq(t, "tags[0].value", tags[0]["value"], "prod")
}

// TestFlattenStorageAccountEmptyTagsAreEmptyList proves an account with no tags
// flattens its tags to a non-nil empty []map[string]interface{}, keeping the
// list block shape stable.
func TestFlattenStorageAccountEmptyTagsAreEmptyList(t *testing.T) {
	got := FlattenStorageAccount(&client.StorageAccount{ID: "acct-empty"})
	tags, ok := got["tags"].([]map[string]interface{})
	if !ok {
		t.Fatalf("tags has type %T, want []map[string]interface{}", got["tags"])
	}
	if tags == nil {
		t.Errorf("tags is a nil slice; expected a non-nil empty list")
	}
	if len(tags) != 0 {
		t.Errorf("tags = %v, want empty", tags)
	}
}

// TestFlattenBucketContentAndShape pins the bucket flatten content, with a
// focus on the ACL-adjacent fields the object-storage datasource exposes:
// versioning (a string, never coerced to a bool), retention_period (int64
// preserved), and the deleted-counters shape. The bucket carries no secret, so
// the focus is the versioning/retention shape rather than a secret guard.
func TestFlattenBucketContentAndShape(t *testing.T) {
	bucket := &client.Bucket{
		ID:                  "bkt-1",
		Name:                "data",
		Namespace:           "ns-1",
		RetentionPeriod:     int64(30),
		Versioning:          "Enabled",
		Endpoint:            "https://s3.example",
		TotalSize:           "1024",
		TotalSizeUnit:       "MiB",
		TotalObjects:        int64(42),
		TotalObjectsDeleted: "3",
		TotalSizeDeleted:    "12",
	}

	got := FlattenBucket(bucket)

	assertEq(t, "id", got["id"], "bkt-1")
	assertEq(t, "name", got["name"], "data")
	assertEq(t, "namespace", got["namespace"], "ns-1")
	assertEq(t, "endpoint", got["endpoint"], "https://s3.example")

	// versioning is a string state value (e.g. "Enabled"/"Suspended"), it must
	// not be coerced to a bool — that would be a schema-mismatch break.
	if _, ok := got["versioning"].(string); !ok {
		t.Errorf("versioning has type %T, want string", got["versioning"])
	}
	assertEq(t, "versioning", got["versioning"], "Enabled")

	// retention_period must preserve the int64 magnitude.
	if rp, ok := got["retention_period"].(int64); !ok {
		t.Errorf("retention_period has type %T, want int64", got["retention_period"])
	} else if rp != 30 {
		t.Errorf("retention_period = %d, want 30", rp)
	}

	if to, ok := got["total_objects"].(int64); !ok {
		t.Errorf("total_objects has type %T, want int64", got["total_objects"])
	} else if to != 42 {
		t.Errorf("total_objects = %d, want 42", to)
	}

	assertEq(t, "total_size", got["total_size"], "1024")
	assertEq(t, "total_size_unit", got["total_size_unit"], "MiB")
	assertEq(t, "total_objects_deleted", got["total_objects_deleted"], "3")
	assertEq(t, "total_size_deleted", got["total_size_deleted"], "12")
}

// TestFlattenBucketKeySetIsExact pins the exact emitted key set so a new key
// that the datasource schema does not declare is caught before it breaks the
// read.
func TestFlattenBucketKeySetIsExact(t *testing.T) {
	got := FlattenBucket(&client.Bucket{ID: "x"})
	want := map[string]bool{
		"id": true, "name": true, "namespace": true, "retention_period": true,
		"versioning": true, "endpoint": true, "total_size": true,
		"total_size_unit": true, "total_objects": true,
		"total_objects_deleted": true, "total_size_deleted": true,
	}
	for k := range got {
		if !want[k] {
			t.Errorf("FlattenBucket emits undeclared key %q", k)
		}
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("FlattenBucket is missing the expected key %q", k)
		}
	}
}
