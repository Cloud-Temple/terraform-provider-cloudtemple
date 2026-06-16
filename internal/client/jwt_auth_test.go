package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// stubRoundTripper lets a test return a crafted *http.Response for any request,
// so the JWT auth path can be exercised without a real server.
type stubRoundTripper struct {
	fn func(*http.Request) (*http.Response, error)
}

func (s stubRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return s.fn(r) }

// errReadCloser is a response body whose Read always fails — it simulates a
// connection dropping while the auth response body is being read.
type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("simulated body read failure") }
func (errReadCloser) Close() error             { return nil }

// TestJWTReturnsErrorWhenAuthBodyReadFails pins #340: a failed read of the auth
// response body must surface as an error (fail closed), never (nil, nil) — which
// the caller would turn into a nil-pointer panic on token.Raw.
func TestJWTReturnsErrorWhenAuthBodyReadFails(t *testing.T) {
	c, err := NewClient(&Config{
		Address: "https://shiva.example",
		Transport: stubRoundTripper{fn: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: errReadCloser{}}, nil
		}},
	})
	require.NoError(t, err)

	require.NotPanics(t, func() {
		token, err := c.JWT(context.Background())
		require.Error(t, err, "a failed auth-body read must surface as an error, not (nil, nil)")
		require.Nil(t, token)
	})
	require.Nil(t, c.SavedToken, "a failed read must not cache a token")
}

// TestJWTReturnsErrorOnUnparseableTokenAndDoesNotCache pins the adjacent defect:
// on a parse error the partially-parsed token must NOT be cached, otherwise the
// next cache hit reads its (malformed) claims and panics. "aaa.bbb.ccc" is a
// 3-segment string whose header fails to decode, so ParseUnverified returns a
// non-nil token AND an error — the exact shape that the old code cached.
func TestJWTReturnsErrorOnUnparseableTokenAndDoesNotCache(t *testing.T) {
	c, err := NewClient(&Config{
		Address: "https://shiva.example",
		Transport: stubRoundTripper{fn: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("aaa.bbb.ccc")),
			}, nil
		}},
	})
	require.NoError(t, err)

	token, err := c.JWT(context.Background())
	require.Error(t, err, "an unparseable auth token must surface as an error")
	require.Nil(t, token)
	require.Nil(t, c.SavedToken, "a parse failure must not cache a bad token")

	// With the bad token cached (old behaviour), this second call panics in the
	// cache-hit branch while reading the malformed claims.
	require.NotPanics(t, func() { _, _ = c.JWT(context.Background()) })
}
