package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

// resp builds a minimal *http.Response with a readable body, as doWithRetry's
// drain path (closeResponseBody) requires a non-nil Body.
func resp(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// timeoutErr is a net error whose Timeout() == true, as produced (wrapped in a
// *url.Error) when http.Client.Timeout fires.
type timeoutErr struct{}

func (timeoutErr) Error() string { return "request timeout" }
func (timeoutErr) Timeout() bool { return true }

// --- doWithRetry: the retry loop in isolation (deterministic, no network) -----

func TestDoWithRetryRetriesTransientStatusThenSucceeds(t *testing.T) {
	c := &Client{readRetryMax: 3, readRetryBackoffBase: 0}
	var calls int32
	r, err := c.doWithRetry(context.Background(), func() (*http.Response, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			return resp(http.StatusBadGateway, "upstream"), nil // intermittent 502
		}
		return resp(http.StatusOK, "[]"), nil
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "a transient 502 must be retried, then succeed")
}

func TestDoWithRetryRetriesTransportErrorThenSucceeds(t *testing.T) {
	c := &Client{readRetryMax: 3, readRetryBackoffBase: 0}
	var calls int32
	r, err := c.doWithRetry(context.Background(), func() (*http.Response, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			return nil, &url.Error{Op: "Get", URL: "http://x", Err: errors.New("connection reset by peer")}
		}
		return resp(http.StatusOK, "[]"), nil
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "a non-timeout transport error must be retried")
}

func TestDoWithRetryStopsAfterReadRetryMax(t *testing.T) {
	c := &Client{readRetryMax: 3, readRetryBackoffBase: 0}
	var calls int32
	r, err := c.doWithRetry(context.Background(), func() (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return resp(http.StatusInternalServerError, "boom"), nil
	})
	// doWithRetry hands the final response to the caller (requireOK turns it into
	// an error); the point here is the retry is BOUNDED and fails closed.
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, r.StatusCode)
	require.Equal(t, int32(3), atomic.LoadInt32(&calls), "a persistent 5xx must stop after exactly readRetryMax attempts")
}

func TestDoWithRetryDoesNotRetryConfiguredTimeout(t *testing.T) {
	c := &Client{readRetryMax: 3, readRetryBackoffBase: 0}
	var calls int32
	r, err := c.doWithRetry(context.Background(), func() (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return nil, &url.Error{Op: "Get", URL: "http://x", Err: timeoutErr{}}
	})
	require.Error(t, err)
	require.Nil(t, r)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "a configured request timeout must NOT be retried (no multi-minute stall)")
}

func TestDoWithRetryDoesNotRetry4xx(t *testing.T) {
	c := &Client{readRetryMax: 3, readRetryBackoffBase: 0}
	for _, code := range []int{http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound, http.StatusConflict} {
		var calls int32
		r, err := c.doWithRetry(context.Background(), func() (*http.Response, error) {
			atomic.AddInt32(&calls, 1)
			return resp(code, ""), nil
		})
		require.NoError(t, err)
		require.Equal(t, code, r.StatusCode)
		require.Equal(t, int32(1), atomic.LoadInt32(&calls), "a %d is definitive and must never be retried", code)
	}
}

func TestDoWithRetryDisabledMakesSingleAttempt(t *testing.T) {
	// A bare Config{} leaves ReadRetryMax == 0 (retry opt-in via DefaultConfig).
	c := &Client{readRetryMax: 0, readRetryBackoffBase: 0}
	var calls int32
	r, err := c.doWithRetry(context.Background(), func() (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return resp(http.StatusBadGateway, ""), nil
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadGateway, r.StatusCode)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "retry disabled must preserve single-shot behaviour")
}

func TestDoWithRetryHonorsContextCancellationDuringBackoff(t *testing.T) {
	c := &Client{readRetryMax: 3, readRetryBackoffBase: time.Hour} // long backoff so ctx wins
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	var calls int32
	_, err := c.doWithRetry(ctx, func() (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return resp(http.StatusBadGateway, ""), nil
	})
	require.Error(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "a cancelled context must abort before a second attempt")
}

func TestRetryableTransportError(t *testing.T) {
	require.False(t, retryableTransportError(nil))
	require.False(t, retryableTransportError(&url.Error{Err: timeoutErr{}}), "configured timeout is not retryable")
	require.False(t, retryableTransportError(context.Canceled))
	require.False(t, retryableTransportError(context.DeadlineExceeded))
	require.True(t, retryableTransportError(&url.Error{Err: errors.New("connection reset by peer")}))
}

func TestParseRetryAfter(t *testing.T) {
	require.Equal(t, time.Duration(0), parseRetryAfter(""))
	require.Equal(t, time.Duration(0), parseRetryAfter("Wed, 21 Oct 2026 07:28:00 GMT")) // HTTP-date ignored (best-effort)
	require.Equal(t, time.Duration(0), parseRetryAfter("0"))
	require.Equal(t, 3*time.Second, parseRetryAfter("3"))
}

// --- end-to-end through doRequest / NewClient (real http server) --------------

// newRetryTestClient builds a client against an httptest server with retry
// ENABLED and zero backoff, and a pre-seeded token so JWT does not hit the server.
func newRetryTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient(&Config{Address: srv.URL})
	require.NoError(t, err)
	c.SavedToken = &jwt.Token{Claims: jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())}}
	c.readRetryMax = 3
	c.readRetryBackoffBase = 0
	return c, srv
}

func TestDoRequestRetriesGETButNotWrites(t *testing.T) {
	ctx := context.Background()

	t.Run("GET retried on 502 then succeeds", func(t *testing.T) {
		var calls int32
		c, _ := newRetryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt32(&calls, 1) == 1 {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		})
		r, err := c.doRequest(ctx, c.newRequest("GET", "/compute/v1/open_iaas"))
		require.NoError(t, err)
		defer closeResponseBody(r)
		require.Equal(t, http.StatusOK, r.StatusCode)
		require.Equal(t, int32(2), atomic.LoadInt32(&calls))
	})

	t.Run("POST not retried (idempotency guard for writes)", func(t *testing.T) {
		var calls int32
		c, _ := newRetryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&calls, 1)
			w.WriteHeader(http.StatusBadGateway)
		})
		r, err := c.doRequest(ctx, c.newRequest("POST", "/some/write"))
		require.NoError(t, err) // no transport error; the 502 response is handed back
		defer closeResponseBody(r)
		require.Equal(t, http.StatusBadGateway, r.StatusCode)
		require.Equal(t, int32(1), atomic.LoadInt32(&calls), "a write must be sent exactly once")
	})
}

func TestNewClientAppliesHTTPTimeout(t *testing.T) {
	t.Run("default applied for a bare Config", func(t *testing.T) {
		c, err := NewClient(&Config{Address: "example.test"})
		require.NoError(t, err)
		require.Equal(t, defaultHTTPTimeout, c.config.HttpClient.Timeout)
	})
	t.Run("explicit Config.HTTPTimeout honored", func(t *testing.T) {
		c, err := NewClient(&Config{Address: "example.test", HTTPTimeout: 7 * time.Second})
		require.NoError(t, err)
		require.Equal(t, 7*time.Second, c.config.HttpClient.Timeout)
	})
}

func TestDefaultConfigReadsHTTPTimeoutEnv(t *testing.T) {
	t.Run("unset keeps default", func(t *testing.T) {
		t.Setenv(HTTPTimeoutEnvName, "")
		require.Equal(t, defaultHTTPTimeout, DefaultConfig().HTTPTimeout)
	})
	t.Run("valid seconds honored", func(t *testing.T) {
		t.Setenv(HTTPTimeoutEnvName, "30")
		require.Equal(t, 30*time.Second, DefaultConfig().HTTPTimeout)
	})
	t.Run("non-numeric keeps default (fail-safe)", func(t *testing.T) {
		t.Setenv(HTTPTimeoutEnvName, "not-a-number")
		require.Equal(t, defaultHTTPTimeout, DefaultConfig().HTTPTimeout)
	})
	t.Run("non-positive keeps default (fail-safe)", func(t *testing.T) {
		t.Setenv(HTTPTimeoutEnvName, "0")
		require.Equal(t, defaultHTTPTimeout, DefaultConfig().HTTPTimeout)
		t.Setenv(HTTPTimeoutEnvName, "-5")
		require.Equal(t, defaultHTTPTimeout, DefaultConfig().HTTPTimeout)
	})
}

func TestHTTPTimeoutFiresFastAndIsNotRetried(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		// Return as soon as the client gives up (timeout) so srv.Close does not block.
		select {
		case <-time.After(5 * time.Second):
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(&Config{Address: srv.URL, HTTPTimeout: 100 * time.Millisecond})
	require.NoError(t, err)
	c.SavedToken = &jwt.Token{Claims: jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())}}
	c.readRetryMax = 3
	c.readRetryBackoffBase = 0

	start := time.Now()
	_, err = c.doRequest(context.Background(), c.newRequest("GET", "/slow"))
	elapsed := time.Since(start)

	require.Error(t, err, "a stuck endpoint must fail, not hang")
	require.Less(t, elapsed, 2*time.Second, "must fail fast at the timeout, not wait for the slow handler")
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "a configured timeout must not be retried")
}

func TestJWTRetriesAuthAndRebuildsBody(t *testing.T) {
	var calls int32
	var mu sync.Mutex
	var bodies []string

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())})
	signed, err := tok.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		mu.Unlock()
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusBadGateway) // transient auth failure
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(signed))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(&Config{Address: srv.URL, ClientID: "id-x", SecretID: "sec-y"})
	require.NoError(t, err)
	c.readRetryMax = 3
	c.readRetryBackoffBase = 0

	token, err := c.JWT(context.Background())
	require.NoError(t, err)
	require.NotNil(t, token)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "the auth POST must be retried on a transient failure")

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, bodies, 2)
	// The crux: the retried request must carry the credentials again — toHTTP
	// consumes r.obj into r.body only once, so without resetting r.body the replay
	// would send an empty body.
	require.Contains(t, bodies[0], "sec-y")
	require.Contains(t, bodies[1], "sec-y", "the rebuilt retry body must still carry the credentials, not an empty/consumed body")
}

// --- the timeout doctrine is uniform across the client (waiters too) ----------

// TestActivityWaitDoesNotRetryConfiguredTimeout pins the systemic guarantee: the
// activity waiter (which polls a read in a retry loop via isTransientAPIError)
// must NOT retry a configured request timeout — otherwise the new per-request
// timeout would be multiplied into a multi-minute stall.
func TestActivityWaitDoesNotRetryConfiguredTimeout(t *testing.T) {
	calls := 0
	timeout := &url.Error{Op: "Get", URL: "https://shiva.example", Err: timeoutErr{}}
	read := scriptedReads(&calls, readOutcome{err: timeout})
	_, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(50), nil)
	require.Error(t, err, "a configured timeout must fail the wait, not be retried")
	require.Equal(t, 1, calls, "a configured request timeout is not retried by the activity waiter")
}

func TestNewClientAppliesTimeoutToProvidedClient(t *testing.T) {
	t.Run("a provided client without a timeout gets the guard", func(t *testing.T) {
		c, err := NewClient(&Config{Address: "example.test", HttpClient: &http.Client{}})
		require.NoError(t, err)
		require.Equal(t, defaultHTTPTimeout, c.config.HttpClient.Timeout)
	})
	t.Run("a provided client with its own timeout is left untouched", func(t *testing.T) {
		c, err := NewClient(&Config{Address: "example.test", HttpClient: &http.Client{Timeout: 9 * time.Second}})
		require.NoError(t, err)
		require.Equal(t, 9*time.Second, c.config.HttpClient.Timeout)
	})
}
