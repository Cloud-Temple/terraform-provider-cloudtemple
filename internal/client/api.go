package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/sethvargo/go-retry"
)

const (
	HTTPAddrEnvName         = "CLOUDTEMPLE_HTTP_ADDR"
	HTTPSchemeEnvName       = "CLOUDTEMPLE_HTTP_SCHEME"
	HTTPClientIDEnvName     = "CLOUDTEMPLE_CLIENT_ID"
	HTTPClientSecretEnvName = "CLOUDTEMPLE_SECRET_ID"
	HTTPTimeoutEnvName      = "CLOUDTEMPLE_HTTP_TIMEOUT"
	FastReadTimeoutEnvName  = "CLOUDTEMPLE_FAST_READ_TIMEOUT"
)

const (
	// defaultHTTPTimeout bounds each HTTP request (dial + headers + body read).
	// It is a safety guard: without it a slow/stuck endpoint hangs a read with no
	// client-side bound. Generous on purpose so it never regresses a legitimately
	// slow request, while still cutting an unbounded hang. Override with
	// CLOUDTEMPLE_HTTP_TIMEOUT (seconds).
	defaultHTTPTimeout = 600 * time.Second

	// defaultFastReadTimeout is the OPT-IN, shorter per-call timeout applied only to
	// designated fast idempotent reads (currently the backup OpenIaaS policies List).
	// Unlike defaultHTTPTimeout, a per-call timeout that fires IS retried (bounded),
	// so it must be short. Override with CLOUDTEMPLE_FAST_READ_TIMEOUT (seconds); an
	// explicit 0 disables it (the read falls back to the global timeout).
	defaultFastReadTimeout = 30 * time.Second

	// defaultReadRetryMax is the total number of attempts for an idempotent GET
	// read (and the auth POST): 1 = no retry. It lets the provider absorb the rare
	// intermittent transient error (e.g. an upstream 502) instead of failing a
	// whole plan/apply on the first one.
	defaultReadRetryMax = 3

	// defaultReadRetryBackoffBase / readRetryBackoffMax bound the wait between
	// read retries (capped exponential: base, 2*base, ... up to the max).
	defaultReadRetryBackoffBase = 500 * time.Millisecond
	readRetryBackoffMax         = 5 * time.Second
)

// errPerCallReadTimeout marks an OPT-IN per-call read timeout: OUR child-context
// deadline expired while the parent (caller) context was still alive. Unlike the
// global http.Client.Timeout — which is never retried — this sentinel IS retried by
// doWithRetry (bounded), which is safe precisely because the per-call timeout is
// short. It is produced ONLY by classifyPerCallTimeout.
var errPerCallReadTimeout = errors.New("per-call read timeout")

type Config struct {
	Address string

	ApiSuffix bool

	Scheme string

	HttpClient *http.Client

	Transport http.RoundTripper

	ClientID, SecretID string

	// HTTPTimeout bounds each HTTP request. 0 means "use the default"
	// (defaultHTTPTimeout); NewClient applies it to the http.Client it builds, and
	// to a caller-supplied HttpClient that has no timeout of its own, so every
	// client carries the hang guard regardless of how Config was created.
	HTTPTimeout time.Duration

	// FastReadTimeout is an OPT-IN, shorter per-call timeout applied ONLY to reads
	// that opt in (currently the backup OpenIaaS policies List). 0 disables it (the
	// read uses the global HTTPTimeout). Unlike HTTPTimeout, a per-call timeout that
	// fires IS retried (bounded). DefaultConfig sets it; NewClient does NOT backfill
	// it, so a bare Config{} keeps it disabled.
	FastReadTimeout time.Duration

	// ReadRetryMax is the total number of attempts for an idempotent GET read (and
	// the auth POST), 1 meaning no retry. DefaultConfig sets it; a zero value
	// disables retry. It is intentionally NOT backfilled by NewClient, so only a
	// DefaultConfig-derived (production) client retries transient read failures — a
	// minimal/bare Config{} keeps single-shot, deterministic behaviour.
	ReadRetryMax int

	// this parameter will only be used during the tests and not exposed to
	// clients
	ErrorOnUnexpectedActivity bool
}

func DefaultConfig() *Config {
	config := &Config{
		Address:         "shiva.cloud-temple.com",
		Scheme:          "https",
		Transport:       cleanhttp.DefaultPooledTransport(),
		HTTPTimeout:     defaultHTTPTimeout,
		FastReadTimeout: defaultFastReadTimeout,
		ReadRetryMax:    defaultReadRetryMax,
	}

	if addr := os.Getenv(HTTPAddrEnvName); addr != "" {
		config.Address = addr
	}

	// CLOUDTEMPLE_HTTP_TIMEOUT overrides the per-request timeout, in seconds. A
	// missing, non-numeric or non-positive value keeps the default (fail-safe):
	// the timeout is a guard, so we never let a bad env value disable it.
	if v := os.Getenv(HTTPTimeoutEnvName); v != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && secs > 0 {
			config.HTTPTimeout = time.Duration(secs) * time.Second
		}
	}

	// CLOUDTEMPLE_FAST_READ_TIMEOUT overrides the per-call fast-read timeout, in
	// seconds. An explicit "0" DISABLES it (the opt-in reads fall back to the global
	// timeout); a missing or non-numeric value keeps the default. Unlike the global
	// timeout, 0 is honoured here — it is an opt-in convenience, not the hang guard.
	if v := os.Getenv(FastReadTimeoutEnvName); v != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && secs >= 0 {
			config.FastReadTimeout = time.Duration(secs) * time.Second
		}
	}

	if scheme := os.Getenv(HTTPSchemeEnvName); scheme != "" {
		config.Scheme = scheme
	}

	if clientID := os.Getenv(HTTPClientIDEnvName); clientID != "" {
		config.ClientID = clientID
	}

	if secretID := os.Getenv(HTTPClientSecretEnvName); secretID != "" {
		config.SecretID = secretID
	}

	return config
}

type BaseObject struct {
	ID   string
	Name string
}

type Client struct {
	lock       sync.Mutex
	SavedToken *jwt.Token

	config Config

	UserAgent string

	// readRetryMax / readRetryBackoffBase tune the bounded retry of idempotent GET
	// reads and the auth POST (see doWithRetry). readRetryMax is carried from
	// Config.ReadRetryMax (0 => no retry); readRetryBackoffBase is set from the
	// constant default. Both are unexported so in-package tests can drive the retry
	// path deterministically (small/zero backoff, explicit attempt count).
	readRetryMax         int
	readRetryBackoffBase time.Duration
}

func NewClient(config *Config) (*Client, error) {
	defConfig := DefaultConfig()

	if config.Address == "" {
		config.Address = defConfig.Address
	}
	if config.HttpClient == nil {
		config.HttpClient = defConfig.HttpClient
	}
	if config.Transport == nil {
		config.Transport = defConfig.Transport
	}
	if config.Scheme == "" {
		config.Scheme = defConfig.Scheme
	}
	// Backfill the per-request timeout for every client (the hang guard must not
	// depend on how Config was built). ReadRetryMax is deliberately NOT backfilled:
	// retry stays opt-in via DefaultConfig (see Config.ReadRetryMax).
	if config.HTTPTimeout <= 0 {
		config.HTTPTimeout = defConfig.HTTPTimeout
	}

	if config.HttpClient == nil {
		config.HttpClient = &http.Client{
			Transport: config.Transport,
			Timeout:   config.HTTPTimeout,
		}
	} else if config.HttpClient.Timeout <= 0 {
		// A caller-supplied client without its own timeout still gets the hang
		// guard; an explicit non-zero timeout is left untouched.
		config.HttpClient.Timeout = config.HTTPTimeout
	}

	parts := strings.SplitN(config.Address, "://", 2)
	if len(parts) == 2 {
		config.Scheme = parts[0]
		config.Address = parts[1]
	}

	return &Client{
		config:               *config,
		readRetryMax:         config.ReadRetryMax,
		readRetryBackoffBase: defaultReadRetryBackoffBase,
	}, nil
}

type request struct {
	config *Config
	method string
	url    *url.URL
	params url.Values
	body   io.Reader
	obj    any

	// timeout, when > 0, is an OPT-IN per-call deadline for this single request
	// (covering headers AND body); a deadline that fires is retried by doWithRetry
	// (bounded). 0 means the request relies only on the global http.Client.Timeout.
	// See doRequestOnce.
	timeout time.Duration
}

func (c *Client) newRequest(method, path string, args ...interface{}) *request {
	if c.config.ApiSuffix {
		path = "/api" + path
	}

	r := &request{
		config: &c.config,
		method: method,
		url: &url.URL{
			Host:   c.config.Address,
			Scheme: c.config.Scheme,
			Path:   fmt.Sprintf(path, args...),
		},
		params: make(map[string][]string),
	}

	return r
}

func (r *request) addFilter(filter any) {
	f := reflect.ValueOf(filter).Elem()
	if !f.IsValid() || f.IsZero() {
		return
	}

	for _, field := range reflect.VisibleFields(f.Type()) {
		name, found := field.Tag.Lookup("filter")
		if !found {
			continue
		}
		field := f.FieldByName(field.Name)
		if !field.IsValid() || field.IsZero() {
			continue
		}
		switch typ := field.Type().String(); typ {
		case "string":
			r.params.Add(name, field.Interface().(string))
		case "*bool":
			r.params.Add(name, strconv.FormatBool(*field.Interface().(*bool)))
		case "bool":
			r.params.Add(name, strconv.FormatBool(field.Interface().(bool)))
		case "[]string":
			stringSlice := field.Interface().([]string)
			for _, s := range stringSlice {
				r.params.Add(name+"[]", s)
			}
		default:
			panic(fmt.Sprintf("unknown type: %q", typ))
		}
	}
}

func (r *request) toHTTP(ctx context.Context, token, userAgent string) (*http.Request, error) {
	r.url.RawQuery = r.params.Encode()

	if r.body == nil && r.obj != nil {
		b, err := encodeBody(r.obj)
		if err != nil {
			return nil, err
		}
		r.body = b
	}

	req, err := http.NewRequestWithContext(ctx, r.method, r.url.RequestURI(), r.body)
	if err != nil {
		return nil, err
	}

	req.URL.Host = r.url.Host
	req.URL.Scheme = r.url.Scheme

	req.Header = make(http.Header)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Content-Type must always be set when a body is present
	if req.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", userAgent)
	}

	return req, nil
}

func (c *Client) JWT(ctx context.Context) (*jwt.Token, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.SavedToken != nil {
		// Serve the cached token only while it stays valid for at least 5 more
		// minutes. Read its expiry defensively: a token whose claims are nil, or
		// whose "exp" is absent or non-numeric, is treated as unusable and falls
		// through to re-authentication — never panicking on the type assertions.
		if claims, ok := c.SavedToken.Claims.(jwt.MapClaims); ok {
			if exp, ok := claims["exp"].(float64); ok {
				if time.Until(time.Unix(int64(exp), 0)) > 5*time.Minute {
					return c.SavedToken, nil
				}
			}
		}
	}

	r := c.newRequest("POST", "/iam/v2/auth/personal_access_token")
	r.obj = map[string]interface{}{
		"id":     c.config.ClientID,
		"secret": c.config.SecretID,
	}
	// Retry the auth POST narrowly on transient failures. It is idempotent here
	// (it returns a token; it creates nothing), so it is safe to replay. r.body is
	// reset to nil before each attempt because toHTTP encodes r.obj into r.body
	// only once — without the reset a retry would resend an already-consumed body.
	// Running under c.lock, this also prevents a re-auth storm from concurrent
	// callers (they queue behind the single in-flight auth).
	resp, err := c.doWithRetry(ctx, func() (*http.Response, error) {
		r.body = nil
		return c.doRequestWithToken(ctx, r, "")
	})
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Surface the read failure (fail closed) instead of returning a nil token
		// with no error — the caller would dereference token.Raw and panic.
		return nil, fmt.Errorf("failed to read authentication response: %w", err)
	}

	token, _, err := new(jwt.Parser).ParseUnverified(string(body), jwt.MapClaims{})
	if err != nil {
		// Do NOT cache a partially-parsed token on error: a later cache hit reads
		// its claims (the "exp" expiry) and would panic on a malformed token.
		return nil, fmt.Errorf("failed to parse authentication token: %w", err)
	}
	c.SavedToken = token

	return token, nil
}

type LoginToken struct {
	t *jwt.Token
}

func (l *LoginToken) UserID() string {
	return l.t.Claims.(jwt.MapClaims)["userId"].(string)
}

func (l *LoginToken) TenantID() string {
	scope := l.t.Claims.(jwt.MapClaims)["scope"].(map[string]interface{})
	return scope["id"].(string)
}

func (l *LoginToken) CompanyID() string {
	return l.t.Claims.(jwt.MapClaims)["companyId"].(string)
}

func (c *Client) Token(ctx context.Context) (*LoginToken, error) {
	token, err := c.JWT(ctx)
	if err != nil {
		return nil, err
	}

	return &LoginToken{
		t: token,
	}, nil

}

// doRequest performs an authenticated request. Idempotent GET reads are retried
// on transient failures (see doWithRetry) so a slow/flaky API does not fail a
// whole plan/apply on the first intermittent error. Non-GET requests (writes,
// which may already have triggered an async activity via the Location header)
// are sent exactly once — retrying them could double-create.
func (c *Client) doRequest(ctx context.Context, r *request) (*http.Response, error) {
	if r.method == http.MethodGet {
		return c.doWithRetry(ctx, func() (*http.Response, error) {
			return c.doRequestOnce(ctx, r)
		})
	}
	return c.doRequestOnce(ctx, r)
}

// classifyPerCallTimeout wraps err in the errPerCallReadTimeout sentinel IFF it is
// OUR per-call child deadline that expired while the parent context is still alive.
// It deliberately does NOT wrap the global http.Client.Timeout (which does not make
// callCtx itself expire) nor a parent cancellation/deadline (parent.Err() != nil) —
// so doWithRetry retries ONLY a genuine per-call timeout, never the global timeout
// nor caller cancellation. Extracted as a pure function so the guard is unit-tested
// directly, independently of the retry loop.
func classifyPerCallTimeout(callCtx, parent context.Context, err error) error {
	if err != nil && callCtx.Err() == context.DeadlineExceeded && parent.Err() == nil {
		return fmt.Errorf("%w: %w", errPerCallReadTimeout, err)
	}
	return err
}

// doRequestOnce performs a single authenticated request, with no retry.
//
// When r.timeout > 0 (an OPT-IN per-call timeout, currently only the backup
// policies List), the request AND its response body are bounded by a child context
// of that duration, and the body is buffered in memory before returning so the
// deadline covers the whole read (a stalled body is bounded too). A per-call
// deadline that fires while the parent context is still alive is wrapped in
// errPerCallReadTimeout so doWithRetry retries it (bounded); the global
// http.Client.Timeout and a parent cancellation/deadline are never wrapped, so they
// are never retried as a timeout. When r.timeout == 0 the path is unchanged (the
// body is streamed to the caller).
func (c *Client) doRequestOnce(ctx context.Context, r *request) (*http.Response, error) {
	token, err := c.JWT(ctx)
	if err != nil {
		return nil, err
	}

	if r.timeout <= 0 {
		resp, err := c.doRequestWithToken(ctx, r, token.Raw)
		if err != nil {
			return nil, err
		}
		if c.config.ErrorOnUnexpectedActivity && resp.Header.Get("Location") != "" {
			closeResponseBody(resp)
			return nil, fmt.Errorf("an unexpected Location header has been found")
		}
		return resp, nil
	}

	// Opt-in per-call timeout: bound headers AND body with a child context. cancel()
	// is safe on return because the body is fully read (buffered) before we return.
	callCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	resp, err := c.doRequestWithToken(callCtx, r, token.Raw)
	if err != nil {
		return nil, classifyPerCallTimeout(callCtx, ctx, err)
	}
	if c.config.ErrorOnUnexpectedActivity && resp.Header.Get("Location") != "" {
		closeResponseBody(resp)
		return nil, fmt.Errorf("an unexpected Location header has been found")
	}

	buf, rerr := io.ReadAll(resp.Body)
	if rerr != nil {
		resp.Body.Close()
		// (a) our own per-call deadline -> sentinel (doWithRetry retries it).
		if wrapped := classifyPerCallTimeout(callCtx, ctx, rerr); errors.Is(wrapped, errPerCallReadTimeout) {
			return nil, wrapped
		}
		// (b) a body-read failure on a transient status -> hand the response back so
		// doWithRetry retries by STATUS, identical to the existing streaming path.
		if transientStatus(resp.StatusCode) {
			resp.Body = io.NopCloser(bytes.NewReader(buf))
			return resp, nil
		}
		// (c) a genuine body error on a non-retryable status -> surface unchanged.
		return nil, rerr
	}
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(buf))
	return resp, nil
}

// doWithRetry runs send up to c.readRetryMax times (1 => no retry), retrying a
// transient status (429 / 5xx, after draining the response body so the keep-alive
// connection is reused), a retryable transport error, or the OPT-IN per-call read
// timeout (errPerCallReadTimeout, produced by doRequestOnce when r.timeout fires),
// with a bounded backoff that honours Retry-After on 429. It NEVER retries the
// GLOBAL request timeout (http.Client.Timeout), a parent context error, a 4xx or a
// decode error — those are returned at once. The last attempt's result is handed to
// the caller, which validates the FINAL response (requireOK / requireNotFoundOrOK)
// and decodes it. The caller guarantees send is idempotent (a GET, or the auth POST
// whose body is rebuilt each attempt).
func (c *Client) doWithRetry(ctx context.Context, send func() (*http.Response, error)) (*http.Response, error) {
	attempts := c.readRetryMax
	if attempts < 1 {
		attempts = 1
	}

	var (
		resp *http.Response
		err  error
	)
	for attempt := 0; attempt < attempts; attempt++ {
		resp, err = send()

		if attempt == attempts-1 {
			return resp, err // budget exhausted: the caller validates the final result
		}

		switch {
		case err != nil:
			if !retryableTransportError(err) && !errors.Is(err, errPerCallReadTimeout) {
				return resp, err
			}
			if !c.waitBeforeRetry(ctx, attempt, 0) {
				return nil, ctx.Err()
			}
		case resp != nil && transientStatus(resp.StatusCode):
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			// Drain+close the failed response before retrying so net/http can reuse
			// the connection instead of opening a new one against a struggling gateway.
			closeResponseBody(resp)
			if !c.waitBeforeRetry(ctx, attempt, retryAfter) {
				return nil, ctx.Err()
			}
		default:
			return resp, err // success or a non-retryable status: let the caller handle it
		}
	}
	return resp, err
}

// waitBeforeRetry sleeps the backoff for the just-failed attempt (or Retry-After,
// capped), honouring ctx. It returns false if ctx is cancelled during the wait.
func (c *Client) waitBeforeRetry(ctx context.Context, attempt int, retryAfter time.Duration) bool {
	wait := c.readRetryBackoff(attempt)
	if retryAfter > 0 {
		if retryAfter > readRetryBackoffMax {
			retryAfter = readRetryBackoffMax
		}
		wait = retryAfter
	}
	if wait <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// readRetryBackoff is the bounded wait before the retry following the 0-based
// failed attempt: a capped exponential (base, 2*base, 4*base, ... up to the max).
func (c *Client) readRetryBackoff(attempt int) time.Duration {
	base := c.readRetryBackoffBase
	if base <= 0 {
		return 0
	}
	d := base << attempt
	if d <= 0 || d > readRetryBackoffMax { // d<=0 guards a shift overflow on a large attempt
		d = readRetryBackoffMax
	}
	return d
}

// transientStatus reports whether an HTTP status is worth retrying for an
// idempotent request: rate limiting (429) or a server-side error (5xx, including
// the 502 the upstream gateway emits intermittently).
func transientStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

// retryableTransportError reports whether a transport-level error from an
// idempotent request is worth retrying. It deliberately EXCLUDES timeouts (the
// configured per-request http.Client.Timeout, or a ctx deadline) and context
// cancellation: retrying a configured timeout would multiply the anti-hang bound
// into a multi-minute stall. Remaining transport failures (connection reset,
// unexpected EOF, connection refused, ...) are transient and retried.
func retryableTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// A configured per-request timeout / deadline reports Timeout() == true and
		// must not be retried; any other transport failure is transient.
		return !urlErr.Timeout()
	}
	return false
}

// parseRetryAfter parses a Retry-After header expressed in delta-seconds. The
// HTTP-date form is intentionally ignored (best-effort). Returns 0 when the header
// is absent or not a positive integer.
func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

func (c *Client) doRequestAndReturnActivity(ctx context.Context, r *request) (string, error) {
	token, err := c.JWT(ctx)
	if err != nil {
		return "", err
	}

	resp, err := c.doRequestWithToken(ctx, r, token.Raw)
	if err != nil {
		return "", err
	}

	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return "", err
	}

	activityId := resp.Header.Get("Location")
	if activityId == "" {
		return "", fmt.Errorf("no activity ID found in response")
	}
	return activityId, nil
}

func (c *Client) doRequestWithToken(ctx context.Context, r *request, token string) (*http.Response, error) {
	req, err := r.toHTTP(ctx, token, c.UserAgent)
	if err != nil {
		return nil, err
	}
	return c.config.HttpClient.Do(req)
}

// closeResponseBody reads resp.Body until EOF, and then closes it. The read
// is necessary to ensure that the http.Client's underlying RoundTripper is able
// to re-use the TCP connection. See godoc on net/http.Client.Do.
func closeResponseBody(resp *http.Response) error {
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.Body.Close()
}

// requireOK is used to wrap doRequest and check for a 200
func requireOK(resp *http.Response) error {
	return requireHttpCodes(resp, 200, 201, 206)
}

// requireHttpCodes checks for the "allowable" http codes for a response
func requireHttpCodes(resp *http.Response, httpCodes ...int) error {
	// if there is an http code that we require, return w no error
	for _, httpCode := range httpCodes {
		if resp.StatusCode == httpCode {
			return nil
		}
	}

	// if we reached here, then none of the http codes in resp matched any that we expected
	// so err out
	return generateUnexpectedResponseCodeError(resp)
}

func requireNotFoundOrOK(resp *http.Response, notFoundCode int) (bool, error) {
	switch resp.StatusCode {
	case 200, 206:
		return true, nil
	case 404, notFoundCode:
		return false, nil
	case 403:
		return false, fmt.Errorf("access denied: %s", resp.Status)
	default:
		return false, generateUnexpectedResponseCodeError(resp)
	}
}

type StatusError struct {
	Code int
	Body string
}

func (e StatusError) Error() string {
	return fmt.Sprintf("Unexpected response code: %d (%s)", e.Code, e.Body)
}

// isTransientAPIError reports whether an error is worth retrying: throttling
// (429), server-side errors (5xx) or a transient transport failure (connection
// reset, unexpected EOF, ...). A CONFIGURED request timeout (http.Client.Timeout)
// or a context deadline/cancellation is NOT transient — retrying it would
// multiply the per-request bound into a multi-minute stall — and neither are
// authentication, authorization or decoding errors. Shared by the read-retry
// helper (doWithRetry) and the activity/backup waiters so the timeout doctrine
// stays uniform across the client.
func isTransientAPIError(err error) bool {
	var statusErr StatusError
	if errors.As(err, &statusErr) {
		return statusErr.Code == http.StatusTooManyRequests || statusErr.Code >= 500
	}
	return retryableTransportError(err)
}

// generateUnexpectedResponseCodeError consumes the rest of the body, closes
// the body stream and generates an error indicating the status code was
// unexpected.
func generateUnexpectedResponseCodeError(resp *http.Response) error {
	var buf bytes.Buffer
	io.Copy(&buf, resp.Body)
	closeResponseBody(resp)

	trimmed := strings.TrimSpace(buf.String())
	return StatusError{Code: resp.StatusCode, Body: trimmed}
}

// decodeBody is used to JSON decode a body
func decodeBody(resp *http.Response, out any) error {
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

// encodeBody is used to encode a request body
func encodeBody(obj any) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf, nil
}

type WaiterOptions struct {
	Logger func(msg string)
}

func (w *WaiterOptions) log(msg string) {
	if w != nil && w.Logger != nil {
		w.Logger(msg)
	}
}

func (w *WaiterOptions) error(err error) error {
	w.log(fmt.Sprintf("got non-retryable error: %s", err.Error()))
	return err
}

func (w *WaiterOptions) retryableError(err error) error {
	w.log(fmt.Sprintf("got retryable error: %s", err.Error()))
	return retry.RetryableError(err)
}
