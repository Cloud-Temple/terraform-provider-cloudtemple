package client

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// realDialTimeoutError produces the error of ticket #527 the way PRODUCTION does:
// through net/http and net, with a resolver that blocks until the DIALER's own
// deadline fires. It is NOT built by hand, and that is the entire point of this file.
//
// The hand-built equivalent in api_resilience_test.go
// (`net.DNSError{Err: "i/o timeout", IsTimeout: true}`) omits the UnwrapErr field,
// which the real stack sets to net.errTimeout. That single missing field is what let
// a broken classification pass its own test: `errors.Is(err, context.DeadlineExceeded)`
// is FALSE on the synthetic error and TRUE on the real one, so the synthetic error
// never exercised the guard that was short-circuiting the fix.
//
// The same trap was recorded on this file once before (#490). Build transport errors
// through the stack that raises them, or the test certifies a shape that does not
// occur.
func realDialTimeoutError(t *testing.T) error {
	t.Helper()
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			<-ctx.Done() // never answers: the dialer deadline is what ends this
			return nil, ctx.Err()
		},
	}
	transport := &http.Transport{
		DialContext: (&net.Dialer{Timeout: 200 * time.Millisecond, Resolver: resolver}).DialContext,
	}
	// .invalid is reserved by RFC 2606 and never resolves; the blocking resolver
	// above means no packet leaves the machine either way.
	_, err := (&http.Client{Transport: transport}).Get("https://shiva.invalid/api/iam/v2/auth/personal_access_token")
	if err == nil {
		t.Fatal("expected the dial to time out")
	}
	return err
}

// TestRetryableTransportErrorOnTheRealDialTimeout is the regression test for the
// defect this commit fixes: a DNS/connect timeout raised by the dialer's deadline was
// classified PERMANENT, so the bounded activity-read retries were never consumed and
// a sub-second DNS blip failed a whole apply — the #527 field failure.
func TestRetryableTransportErrorOnTheRealDialTimeout(t *testing.T) {
	err := realDialTimeoutError(t)

	// First, pin the SHAPE. If a future Go release stops wrapping this way, the
	// assertions below would still pass for the wrong reason, so state the shape
	// explicitly and let it fail loudly instead.
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatalf("expected a *url.Error, got %T: %v", err, err)
	}
	if !urlErr.Timeout() {
		t.Fatalf("expected the error to report Timeout(), got %v", err)
	}
	var dnsErr *net.DNSError
	if !errors.As(err, &dnsErr) || !dnsErr.IsTimeout {
		t.Fatalf("expected a *net.DNSError with IsTimeout, got %v", err)
	}

	// This is the property the hand-built error could not reproduce, and the reason
	// the ordering inside retryableTransportError matters.
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("the real dial timeout no longer satisfies errors.Is(context.DeadlineExceeded) — " +
			"re-check the ordering in retryableTransportError, which exists because it DOES")
	}

	if !networkStackTimeout(err) {
		t.Fatalf("networkStackTimeout must recognise a real dial timeout: %v", err)
	}
	if !retryableTransportError(err) {
		t.Fatal("a DNS/connect timeout raised by the dialer deadline MUST be retryable. " +
			"It is the #527 field error: classifying it permanent short-circuits the bounded " +
			"activity-read retries and fails the apply on the first blip, leaving the virtual " +
			"machine outside the state")
	}
	if !isTransientAPIError(err) {
		t.Fatal("isTransientAPIError must follow retryableTransportError for this shape: " +
			"it is what the activity waiter consults")
	}
}

// TestConfiguredRequestTimeoutStaysPermanent guards the other side of the ordering:
// moving networkStackTimeout first must NOT make the client's own configured deadline
// retryable, or the anti-hang bound turns into a multi-minute stall.
func TestConfiguredRequestTimeoutStaysPermanent(t *testing.T) {
	// A server that never answers, with the deadline on the http.Client itself —
	// net/http replaces the underlying error with its own *httpError, which carries
	// neither *net.OpError nor *net.DNSError.
	block := make(chan struct{})
	defer close(block)
	srv := newBlockingServer(t, block)

	_, err := (&http.Client{Timeout: 150 * time.Millisecond}).Get(srv)
	if err == nil {
		t.Fatal("expected the configured client timeout to fire")
	}
	if networkStackTimeout(err) {
		t.Fatalf("the configured request deadline must not look like a network-stack timeout: %v", err)
	}
	if retryableTransportError(err) {
		t.Fatalf("the configured request deadline must stay permanent: %v", err)
	}
}

// newBlockingServer starts a server that accepts the request and never answers until
// block is closed, so the caller's own timeout is what ends the request.
func newBlockingServer(t *testing.T, block <-chan struct{}) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}
