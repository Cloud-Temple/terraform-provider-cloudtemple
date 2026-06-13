package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// newPATTestClient wires a Client to a stub HTTP server and pre-seeds a
// far-future JWT so JWT() never hits the network: the tests exercise
// ListStrict's status handling, not the auth plumbing.
func newPATTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := NewClient(&Config{Address: srv.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SavedToken = &jwt.Token{
		Claims: jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())},
	}
	return c
}

// TestPATListStrict pins the evidence channel: only a complete HTTP 200 is a
// usable listing. A 206 is partial and cannot prove an absence; 201/403/5xx
// are not evidence either; a malformed 200 is an error (#281).
func TestPATListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 hits the right endpoint and returns the parsed tokens", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/iam/v2/personal_access_tokens" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"pat-1","name":"a"},{"id":"pat-2","name":"b"}]`))
		})
		tokens, err := c.IAM().PAT().ListStrict(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tokens) != 2 || tokens[0].ID != "pat-1" || tokens[1].ID != "pat-2" {
			t.Fatalf("unexpected tokens: %+v", tokens)
		}
	})

	t.Run("200 with malformed JSON errors", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not json`))
		})
		if _, err := c.IAM().PAT().ListStrict(ctx); err == nil {
			t.Fatal("a malformed 200 body must return a decode error")
		}
	})

	t.Run("200 with an empty body errors", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			// no body
		})
		if _, err := c.IAM().PAT().ListStrict(ctx); err == nil {
			t.Fatal("an empty 200 body must fail closed (decode error), not be read as an empty listing")
		}
	})

	t.Run("200 with the wrong JSON shape errors", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"pat-1"}`)) // object instead of array
		})
		if _, err := c.IAM().PAT().ListStrict(ctx); err == nil {
			t.Fatal("a 200 that is not a token array must return a decode error")
		}
	})

	for _, code := range []int{
		http.StatusCreated,             // 201
		http.StatusPartialContent,      // 206
		http.StatusForbidden,           // 403
		http.StatusInternalServerError, // 500
	} {
		code := code
		t.Run(http.StatusText(code)+" is rejected", func(t *testing.T) {
			c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				_, _ = w.Write([]byte(`[]`))
			})
			if _, err := c.IAM().PAT().ListStrict(ctx); err == nil {
				t.Fatalf("status %d must be rejected as non-200 evidence", code)
			}
		})
	}
}
