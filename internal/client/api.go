package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/dgrijalva/jwt-go"
	"github.com/hashicorp/go-cleanhttp"
)

type Config struct {
	Address string

	Scheme string

	HttpClient *http.Client

	Transport *http.Transport

	ClientID, SecretID string
}

func DefaultConfig() *Config {
	return &Config{
		Address:   "pp-shiva.cloud-temple.com",
		Scheme:    "https",
		Transport: cleanhttp.DefaultPooledTransport(),
	}
}

type Client struct {
	lock       sync.Mutex
	savedToken *jwt.Token

	config Config
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

	if config.HttpClient == nil {
		config.HttpClient = &http.Client{
			Transport: config.Transport,
		}
	}

	parts := strings.SplitN(config.Address, "://", 2)
	if len(parts) == 2 {
		config.Scheme = parts[0]
		config.Address = parts[1]
	}

	return &Client{config: *config}, nil
}

type request struct {
	config *Config
	method string
	url    *url.URL
	params url.Values
	body   io.Reader
	// header http.Header
	obj interface{}
}

func (c *Client) newRequest(method, path string) *request {
	r := &request{
		config: &c.config,
		method: method,
		url: &url.URL{
			Host:   c.config.Address,
			Scheme: c.config.Scheme,
			Path:   path,
		},
		params: make(map[string][]string),
	}

	return r
}

func (r *request) toHTTP(ctx context.Context, token string) (*http.Request, error) {
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
	// See https://github.com/hashicorp/consul/issues/10011
	if req.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *Client) token(ctx context.Context) (*jwt.Token, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.savedToken != nil {
		return c.savedToken, nil
	}

	r := c.newRequest("POST", "/api/iam/v2/auth/personal_access_token")
	r.obj = map[string]interface{}{
		"id":     c.config.ClientID,
		"secret": c.config.SecretID,
	}
	resp, err := c.doRequestWithToken(ctx, r, "")
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}

	token, _, err := new(jwt.Parser).ParseUnverified(string(bytes), jwt.MapClaims{})
	c.savedToken = token

	return token, err
}

func (c *Client) doRequest(ctx context.Context, r *request) (*http.Response, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	return c.doRequestWithToken(ctx, r, token.Raw)
}

func (c *Client) doRequestWithToken(ctx context.Context, r *request, token string) (*http.Response, error) {
	req, err := r.toHTTP(ctx, token)
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
	return requireHttpCodes(resp, 200)
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

type StatusError struct {
	Code int
	Body string
}

func (e StatusError) Error() string {
	return fmt.Sprintf("Unexpected response code: %d (%s)", e.Code, e.Body)
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
func decodeBody(resp *http.Response, out interface{}) error {
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

// encodeBody is used to encode a request body
func encodeBody(obj interface{}) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf, nil
}
