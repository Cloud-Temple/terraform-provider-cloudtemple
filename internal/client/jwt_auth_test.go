package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
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

// signedJWT builds a parseable JWT carrying the given claims.
func signedJWT(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	require.NoError(t, err)
	return s
}

// authStub returns a transport that answers every request with a 200 carrying a
// fresh valid token, counting the calls (so a test can assert re-auth happened).
func authStub(token string, calls *int32) stubRoundTripper {
	return stubRoundTripper{fn: func(*http.Request) (*http.Response, error) {
		atomic.AddInt32(calls, 1)
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(token))}, nil
	}}
}

// TestJWTCacheHitGuardsExpClaim pins #342: the cache-hit branch must read "exp"
// defensively. A valid far-future token is served from cache; a cached token
// with nil/empty/non-numeric claims must NOT panic — it falls through to a
// single re-authentication.
func TestJWTCacheHitGuardsExpClaim(t *testing.T) {
	fresh := signedJWT(t, jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())})

	t.Run("far-future exp is served from cache without re-auth", func(t *testing.T) {
		var calls int32
		c, err := NewClient(&Config{Address: "https://shiva.example", Transport: authStub(fresh, &calls)})
		require.NoError(t, err)
		c.SavedToken = &jwt.Token{Claims: jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())}}

		tok, err := c.JWT(context.Background())
		require.NoError(t, err)
		require.NotNil(t, tok)
		require.Equal(t, int32(0), atomic.LoadInt32(&calls), "a valid far-future cached token must be served without re-auth")
	})

	for _, bc := range []struct {
		name string
		tok  *jwt.Token
	}{
		{"nil claims", &jwt.Token{}},
		{"non-MapClaims type", &jwt.Token{Claims: jwt.RegisteredClaims{}}},
		{"empty claims (no exp)", &jwt.Token{Claims: jwt.MapClaims{}}},
		{"non-numeric exp", &jwt.Token{Claims: jwt.MapClaims{"exp": "soon"}}},
	} {
		bc := bc
		t.Run("falls through to re-auth without panic on "+bc.name, func(t *testing.T) {
			var calls int32
			c, err := NewClient(&Config{Address: "https://shiva.example", Transport: authStub(fresh, &calls)})
			require.NoError(t, err)
			c.SavedToken = bc.tok

			require.NotPanics(t, func() {
				tok, err := c.JWT(context.Background())
				require.NoError(t, err)
				require.NotNil(t, tok)
			})
			require.Equal(t, int32(1), atomic.LoadInt32(&calls), "an unusable cached token must trigger exactly one re-auth")
		})
	}
}
