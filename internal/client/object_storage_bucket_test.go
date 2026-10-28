package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"
)

// captureBucketWrite records the method, path and decoded JSON body of a bucket
// write, and answers like the API: 201 with the activity id in Location.
func captureBucketWrite(t *testing.T, gotMethod, gotPath *string, gotBody *map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*gotMethod = r.Method
		*gotPath = r.URL.Path
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if err := json.Unmarshal(raw, gotBody); err != nil {
			t.Errorf("body is not a JSON object: %v (%s)", err, raw)
		}
		w.Header().Set("Location", "activity-id")
		w.WriteHeader(http.StatusCreated)
	}
}

// TestBucketWhitelistBody pins the body sent for a bucket whitelist: the CIDRs
// travel as `ipWhiteList` entries carrying only `cidr`. The deprecated
// `whitelist` field is being removed from the API, and a `description` key —
// even empty — would overwrite the description stored for that range.
func TestBucketWhitelistBody(t *testing.T) {
	ctx := context.Background()
	cidrs := []string{"10.0.0.0/8", "192.168.1.0/24"}
	wantEntries := []any{
		map[string]any{"cidr": "10.0.0.0/8"},
		map[string]any{"cidr": "192.168.1.0/24"},
	}

	t.Run("create sends ipWhiteList and never whitelist", func(t *testing.T) {
		var method, path string
		body := map[string]any{}
		c := newPATTestClient(t, captureBucketWrite(t, &method, &path, &body))
		if _, err := c.ObjectStorage().Bucket().Create(ctx, &CreateBucketRequest{
			Name:        "bucket-1",
			AccessType:  "custom",
			IPWhiteList: NewIPWhiteList(cidrs),
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if method != http.MethodPost || path != "/storage/object/v1/buckets" {
			t.Fatalf("unexpected request: %s %s", method, path)
		}
		want := map[string]any{"name": "bucket-1", "accessType": "custom", "ipWhiteList": wantEntries}
		if !reflect.DeepEqual(body, want) {
			t.Fatalf("unexpected body:\n got  %v\n want %v", body, want)
		}
	})

	t.Run("update sends ipWhiteList and never whitelist", func(t *testing.T) {
		var method, path string
		body := map[string]any{}
		c := newPATTestClient(t, captureBucketWrite(t, &method, &path, &body))
		if _, err := c.ObjectStorage().Bucket().UpdateWhitelist(ctx, "bucket-1", &UpdateWhitelistRequest{
			AccessType:  "custom",
			IPWhiteList: NewIPWhiteList(cidrs),
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if method != http.MethodPut || path != "/storage/object/v1/buckets/bucket-1/whitelist" {
			t.Fatalf("unexpected request: %s %s", method, path)
		}
		want := map[string]any{"accessType": "custom", "ipWhiteList": wantEntries}
		if !reflect.DeepEqual(body, want) {
			t.Fatalf("unexpected body:\n got  %v\n want %v", body, want)
		}
	})

	// The API refuses an empty ipWhiteList, so a private or public bucket must
	// send no list at all rather than `[]` or `null`.
	t.Run("no CIDR sends no list on create", func(t *testing.T) {
		var method, path string
		body := map[string]any{}
		c := newPATTestClient(t, captureBucketWrite(t, &method, &path, &body))
		if _, err := c.ObjectStorage().Bucket().Create(ctx, &CreateBucketRequest{
			Name:        "bucket-1",
			AccessType:  "private",
			IPWhiteList: NewIPWhiteList([]string{}),
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]any{"name": "bucket-1", "accessType": "private"}
		if !reflect.DeepEqual(body, want) {
			t.Fatalf("unexpected body:\n got  %v\n want %v", body, want)
		}
	})

	t.Run("no CIDR sends no list on update", func(t *testing.T) {
		var method, path string
		body := map[string]any{}
		c := newPATTestClient(t, captureBucketWrite(t, &method, &path, &body))
		if _, err := c.ObjectStorage().Bucket().UpdateWhitelist(ctx, "bucket-1", &UpdateWhitelistRequest{
			AccessType:  "private",
			IPWhiteList: NewIPWhiteList([]string{}),
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]any{"accessType": "private"}
		if !reflect.DeepEqual(body, want) {
			t.Fatalf("unexpected body:\n got  %v\n want %v", body, want)
		}
	})
}
