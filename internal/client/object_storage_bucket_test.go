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

// TestBucketReadWhitelist pins the evidence channel of the acceptance tests:
// the whitelist is read from GET /buckets/{bucket}/whitelist with
// `includeDescriptions=true` — the only form in which the API answers entries
// (observed on the recette API: without it the answer is a bare list of CIDR
// strings) — decoded entry by entry, and only a complete HTTP 200 counts: a
// 206 is a partial listing that cannot prove what the platform holds, so it
// is an error, not a shorter list.
func TestBucketReadWhitelist(t *testing.T) {
	ctx := context.Background()

	serve := func(status int, body string) (*string, *string, *string, *Client) {
		var method, path, query string
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			method = r.Method
			path = r.URL.Path
			query = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		})
		return &method, &path, &query, c
	}

	t.Run("decodes the entries of a complete answer", func(t *testing.T) {
		method, path, query, c := serve(http.StatusOK, `[{"cidr":"10.0.0.0/8","description":"label","source":"client"},{"cidr":"192.168.1.0/24","description":"","source":"client"}]`)
		got, err := c.ObjectStorage().Bucket().ReadWhitelist(ctx, "bucket-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *method != http.MethodGet || *path != "/storage/object/v1/buckets/bucket-1/whitelist" || *query != "includeDescriptions=true" {
			t.Fatalf("unexpected request: %s %s?%s", *method, *path, *query)
		}
		want := []*BucketWhitelistEntry{{CIDR: "10.0.0.0/8", Description: "label", Source: "client"}, {CIDR: "192.168.1.0/24", Source: "client"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected entries:\n got  %+v\n want %+v", got, want)
		}
	})

	t.Run("a partial answer is an error, not a shorter list", func(t *testing.T) {
		_, _, _, c := serve(http.StatusPartialContent, `[{"cidr":"10.0.0.0/8","source":"client"}]`)
		got, err := c.ObjectStorage().Bucket().ReadWhitelist(ctx, "bucket-1")
		if err == nil {
			t.Fatalf("expected an error on HTTP 206, got entries %+v", got)
		}
	})

	t.Run("an empty whitelist is an empty list", func(t *testing.T) {
		_, _, _, c := serve(http.StatusOK, `[]`)
		got, err := c.ObjectStorage().Bucket().ReadWhitelist(ctx, "bucket-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no entry, got %+v", got)
		}
	})
}
