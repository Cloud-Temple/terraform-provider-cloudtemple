package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

type fakeRoundTripper struct {
	fn func(*http.Request) (*http.Response, error)
}

func (f fakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f.fn(r) }

// TestLoggingHttpTransportRedactsSecretBodyValues proves that a credential
// carried in an HTTP body never reaches the TF_LOG output, even for endpoints
// whose whole body is not masked (object storage returns secretAccessKey; VM
// customization sends a password). Guard under test: the
// MaskAllFieldValuesRegexes(sensitiveBodyValueRegex) call in RoundTrip, with a
// JSON-string-aware value matcher. Each case asserts a distinctive marker that
// must NOT appear in the captured log. Mutation-proven RED:
//   - removing the mask call leaks every marker;
//   - reverting the value matcher to `[^"]*` leaks the escaped-quote case's
//     marker (the suffix after the `\"`).
func TestLoggingHttpTransportRedactsSecretBodyValues(t *testing.T) {
	cases := []struct {
		name        string
		path        string
		reqBody     string
		respBody    string
		mustNotLeak string
	}{
		{
			name:        "object storage secretAccessKey in the response body",
			path:        "/api/storage/object/v1/storage_accounts",
			respBody:    `{"accessKeyId":"AKIAEXAMPLE","secretAccessKey":"S3cret-9f3a-do-not-log"}`,
			mustNotLeak: "S3cret-9f3a-do-not-log",
		},
		{
			name:        "VM password in the request body",
			path:        "/api/compute/v1/virtual_machines",
			reqBody:     `{"name":"vm-1","password":"S3cret-9f3a-do-not-log"}`,
			mustNotLeak: "S3cret-9f3a-do-not-log",
		},
		{
			name:        "global access key in the response body (case-insensitive)",
			path:        "/api/storage/object/v1/namespaces/access_key/renew",
			respBody:    `{"AccessSecretKey":"S3cret-9f3a-do-not-log"}`,
			mustNotLeak: "S3cret-9f3a-do-not-log",
		},
		{
			// JSON value is  pre"TAIL-do-not-log  (an embedded, escaped quote).
			// A naive `[^"]*` matcher stops at the `\"` and leaves "TAIL-do-not-log"
			// visible; the JSON-string-aware matcher masks the whole value.
			name:        "password value containing an escaped JSON quote",
			path:        "/api/compute/v1/virtual_machines",
			reqBody:     `{"name":"vm-1","password":"pre\"TAIL-do-not-log"}`,
			mustNotLeak: "TAIL-do-not-log",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := tflogtest.RootLogger(context.Background(), &buf)

			downstream := fakeRoundTripper{fn: func(r *http.Request) (*http.Response, error) {
				if r.Body != nil {
					_, _ = io.ReadAll(r.Body)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Proto:      "HTTP/1.1",
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tc.respBody)),
					Request:    r,
				}, nil
			}}

			tr := &loggingHttpTransport{transport: logging.NewLoggingHTTPTransport(downstream)}

			var body io.Reader
			if tc.reqBody != "" {
				body = strings.NewReader(tc.reqBody)
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://example.test"+tc.path, body)
			if err != nil {
				t.Fatalf("new request: %s", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip: %s", err)
			}
			// Functional passthrough: the response body must still be readable, intact.
			got, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if tc.respBody != "" && string(got) != tc.respBody {
				t.Fatalf("response body altered: got %q want %q", string(got), tc.respBody)
			}

			out := buf.String()
			if strings.Contains(out, tc.mustNotLeak) {
				t.Fatalf("SECRET LEAKED (%q) in TF_LOG output for %q:\n%s", tc.mustNotLeak, tc.path, out)
			}
			if !strings.Contains(out, "***") {
				t.Fatalf("expected a redaction marker (***) in the log output for %q; masking did not run:\n%s", tc.path, out)
			}
		})
	}
}
