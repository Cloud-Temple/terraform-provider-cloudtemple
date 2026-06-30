package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

// These tests pin the OPT-IN per-GET read timeout with retry-on-timeout (applied
// only to the backup OpenIaaS policies List). The design and its safety properties
// were reviewed adversarially (Codex, 5 plan rounds):
//   - only OUR per-call child deadline (parent still alive) is retried,
//   - the global http.Client.Timeout and a parent cancellation/deadline are NEVER
//     retried,
//   - a stalled BODY (not just headers) is bounded and retried,
//   - a transient-status response whose body read fails keeps its status retry,
//   - non-opt-in calls are unchanged.

// --- the guard, unit-tested directly (independent of the retry loop) ----------

// TestClassifyPerCallTimeout pins classifyPerCallTimeout: it wraps the sentinel
// IFF our child deadline expired AND the parent is still alive. Mutation: dropping
// the `parent.Err()==nil` clause makes the "parent deadline" case wrongly return the
// sentinel -> the sub-test below goes RED. This kills the guard mutation directly,
// without depending on retry timing.
func TestClassifyPerCallTimeout(t *testing.T) {
	boom := errors.New("boom")

	t.Run("our child deadline + parent alive -> sentinel", func(t *testing.T) {
		parent := context.Background()
		child, cancel := context.WithTimeout(parent, time.Nanosecond)
		defer cancel()
		<-child.Done() // child is now DeadlineExceeded
		err := classifyPerCallTimeout(child, parent, boom)
		require.True(t, errors.Is(err, errPerCallReadTimeout), "our child timeout (parent alive) must be the sentinel")
	})

	t.Run("parent deadline propagated to child -> NOT sentinel", func(t *testing.T) {
		parent, pcancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer pcancel()
		<-parent.Done() // parent is DeadlineExceeded
		child, cancel := context.WithTimeout(parent, time.Hour)
		defer cancel()
		<-child.Done() // child inherits the parent's DeadlineExceeded
		err := classifyPerCallTimeout(child, parent, boom)
		require.False(t, errors.Is(err, errPerCallReadTimeout), "a parent deadline must NEVER be wrapped as our per-call timeout")
		require.Equal(t, boom, err)
	})

	t.Run("parent canceled -> NOT sentinel", func(t *testing.T) {
		parent, pcancel := context.WithCancel(context.Background())
		pcancel() // Canceled, not DeadlineExceeded
		child, cancel := context.WithTimeout(parent, time.Hour)
		defer cancel()
		<-child.Done()
		err := classifyPerCallTimeout(child, parent, boom)
		require.False(t, errors.Is(err, errPerCallReadTimeout), "a parent cancellation must never be retried as our timeout")
	})

	t.Run("non-deadline error, both alive -> unchanged", func(t *testing.T) {
		parent := context.Background()
		child, cancel := context.WithCancel(parent)
		defer cancel()
		err := classifyPerCallTimeout(child, parent, boom)
		require.Equal(t, boom, err)
		require.False(t, errors.Is(err, errPerCallReadTimeout))
	})

	t.Run("nil error -> nil", func(t *testing.T) {
		require.NoError(t, classifyPerCallTimeout(context.Background(), context.Background(), nil))
	})
}

// --- doWithRetry: the sentinel is retried, the global timeout is not -----------

// TestDoWithRetryRetriesPerCallTimeoutSentinel pins the doWithRetry predicate
// change. Mutation: removing the `errors.Is(err, errPerCallReadTimeout)` clause
// makes attempt 1's sentinel fatal -> count==1, RED.
func TestDoWithRetryRetriesPerCallTimeoutSentinel(t *testing.T) {
	c := &Client{readRetryMax: 3, readRetryBackoffBase: 0}
	var calls int32
	r, err := c.doWithRetry(context.Background(), func() (*http.Response, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			return nil, fmt.Errorf("%w: simulated body stall", errPerCallReadTimeout)
		}
		return resp(http.StatusOK, "[]"), nil
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "a per-call read timeout must be retried, then succeed")
}

func TestDoWithRetryStopsPerCallTimeoutAfterMax(t *testing.T) {
	c := &Client{readRetryMax: 3, readRetryBackoffBase: 0}
	var calls int32
	r, err := c.doWithRetry(context.Background(), func() (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return nil, fmt.Errorf("%w: persistent stall", errPerCallReadTimeout)
	})
	require.Error(t, err)
	require.Nil(t, r)
	require.True(t, errors.Is(err, errPerCallReadTimeout))
	require.Equal(t, int32(3), atomic.LoadInt32(&calls), "a persistent per-call timeout must stop after exactly readRetryMax attempts")
}

// --- end-to-end through the /policies List (opt-in wired here) -----------------

// newPolicyTimeoutClient wires a Client to a stub server with a deterministic retry
// path (zero backoff) and a pre-seeded far-future JWT. cfg.Address is overwritten.
func newPolicyTimeoutClient(t *testing.T, cfg Config, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg.Address = srv.URL
	if cfg.ReadRetryMax == 0 {
		cfg.ReadRetryMax = 3
	}
	c, err := NewClient(&cfg)
	require.NoError(t, err)
	c.readRetryBackoffBase = 0 // deterministic + fast
	c.SavedToken = &jwt.Token{Claims: jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())}}
	return c
}

// stallUntilGivenUp blocks until the per-call timeout cancels the request, so the
// stub never lingers past the client (httptest.Close stays instant).
func stallUntilGivenUp(r *http.Request) {
	select {
	case <-time.After(10 * time.Second):
	case <-r.Context().Done():
	}
}

func TestPoliciesListStalledHeadersRetriesThenSucceeds(t *testing.T) {
	var calls int32
	c := newPolicyTimeoutClient(t, Config{FastReadTimeout: 150 * time.Millisecond}, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			stallUntilGivenUp(r) // first attempt hangs past the per-call timeout
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	})
	_, err := c.Backup().OpenIaaS().Policy().List(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "a stalled-headers attempt must time out, retry, and succeed")
}

func TestPoliciesListStalledBodyRetriesThenSucceeds(t *testing.T) {
	var calls int32
	c := newPolicyTimeoutClient(t, Config{FastReadTimeout: 150 * time.Millisecond}, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusOK)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			_, _ = io.WriteString(w, "[") // partial body, never completed
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			stallUntilGivenUp(r) // body stalls past the per-call timeout
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	})
	_, err := c.Backup().OpenIaaS().Policy().List(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "a stalled BODY must time out inside the read, retry, and succeed")
}

func TestPoliciesListAlwaysSlowFailsAfterMax(t *testing.T) {
	var calls int32
	c := newPolicyTimeoutClient(t, Config{FastReadTimeout: 120 * time.Millisecond}, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		stallUntilGivenUp(r)
	})
	_, err := c.Backup().OpenIaaS().Policy().List(context.Background(), nil)
	require.Error(t, err)
	require.Equal(t, int32(3), atomic.LoadInt32(&calls), "a persistently slow endpoint must fail after exactly readRetryMax attempts, not wait the global timeout")
}

func TestPoliciesListSuccessDecodes(t *testing.T) {
	c := newPolicyTimeoutClient(t, Config{FastReadTimeout: 2 * time.Second}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"p1","name":"daily"}]`))
	})
	policies, err := c.Backup().OpenIaaS().Policy().List(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, policies, 1)
	require.Equal(t, "p1", policies[0].ID)
}

// TestPoliciesListGlobalTimeoutNotRetried: when the GLOBAL http.Client.Timeout is
// shorter than the per-call timeout, the global timeout fires first and must NOT be
// retried (it is not our child deadline).
func TestPoliciesListGlobalTimeoutNotRetried(t *testing.T) {
	var calls int32
	c := newPolicyTimeoutClient(t, Config{HTTPTimeout: 100 * time.Millisecond, FastReadTimeout: 5 * time.Second}, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		stallUntilGivenUp(r)
	})
	_, err := c.Backup().OpenIaaS().Policy().List(context.Background(), nil)
	require.Error(t, err)
	require.False(t, errors.Is(err, errPerCallReadTimeout), "the global timeout must not be classified as our per-call timeout")
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "the global http.Client.Timeout must never be retried")
}

// TestPoliciesListTransientStatusBrokenBodyStatusRetry: a 502 whose body read fails
// (truncated) must keep the existing status retry (return resp,nil so doWithRetry
// retries by status), not be lost as a non-retryable body error.
func TestPoliciesListTransientStatusBrokenBodyStatusRetry(t *testing.T) {
	var calls int32
	c := newPolicyTimeoutClient(t, Config{FastReadTimeout: 2 * time.Second}, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Error("server does not support hijacking")
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			// Declare a longer body than is written, then close: the client's body
			// read fails with io.ErrUnexpectedEOF on a transient (502) status.
			_, _ = io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 100\r\n\r\nshort")
			_ = conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	})
	_, err := c.Backup().OpenIaaS().Policy().List(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "a transient 502 with a broken body must still be retried by status")
}

// TestPoliciesListOptOutNotRetried: with FastReadTimeout==0 the per-call timeout is
// disabled (the streaming path), so a slow endpoint is bounded only by the global
// timeout and is NOT retried — proving the opt-in is isolated.
func TestPoliciesListOptOutNotRetried(t *testing.T) {
	var calls int32
	c := newPolicyTimeoutClient(t, Config{HTTPTimeout: 100 * time.Millisecond, FastReadTimeout: 0}, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		stallUntilGivenUp(r)
	})
	_, err := c.Backup().OpenIaaS().Policy().List(context.Background(), nil)
	require.Error(t, err)
	require.False(t, errors.Is(err, errPerCallReadTimeout))
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "with the per-call timeout disabled, the global timeout is not retried")
}
