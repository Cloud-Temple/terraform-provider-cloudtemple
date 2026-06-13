package client

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestBucketListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 returns the parsed buckets at the right endpoint", func(t *testing.T) {
		var method, path string
		var query url.Values
		c := newPATTestClient(t, captureHandler(http.StatusOK, `[{"id":"b1","name":"bucket-1"},{"id":"b2","name":"bucket-2"}]`, &method, &path, &query))
		buckets, err := c.ObjectStorage().Bucket().ListStrict(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(buckets) != 2 || buckets[0].Name != "bucket-1" {
			t.Fatalf("unexpected buckets: %+v", buckets)
		}
		if method != http.MethodGet || path != "/storage/object/v1/buckets" {
			t.Fatalf("unexpected request: %s %s", method, path)
		}
	})

	t.Run("an empty array is a valid empty listing", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		})
		buckets, err := c.ObjectStorage().Bucket().ListStrict(ctx)
		if err != nil || len(buckets) != 0 {
			t.Fatalf("an empty array must be accepted, got buckets=%v err=%v", buckets, err)
		}
	})

	t.Run("200 with the wrong JSON shape errors", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"b1"}`)) // object instead of array
		})
		if _, err := c.ObjectStorage().Bucket().ListStrict(ctx); err == nil {
			t.Fatal("a 200 that is not a bucket array must return a decode error")
		}
	})

	runListStrictRejections(t, func(c *Client) error {
		_, err := c.ObjectStorage().Bucket().ListStrict(ctx)
		return err
	})
}

func TestStorageAccountListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 returns the parsed accounts at the right endpoint", func(t *testing.T) {
		var method, path string
		var query url.Values
		c := newPATTestClient(t, captureHandler(http.StatusOK, `[{"id":"a1","name":"acct-1"}]`, &method, &path, &query))
		accounts, err := c.ObjectStorage().StorageAccount().ListStrict(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(accounts) != 1 || accounts[0].Name != "acct-1" {
			t.Fatalf("unexpected accounts: %+v", accounts)
		}
		if method != http.MethodGet || path != "/storage/object/v1/storage_accounts" {
			t.Fatalf("unexpected request: %s %s", method, path)
		}
	})

	runListStrictRejections(t, func(c *Client) error {
		_, err := c.ObjectStorage().StorageAccount().ListStrict(ctx)
		return err
	})
}
