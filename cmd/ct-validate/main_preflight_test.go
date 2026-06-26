package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestResolveTarget pins that resolveTarget reports the scheme/host the client
// will ACTUALLY use, including the "scheme://" prefix override that NewClient
// applies (api.go) — the exact path by which an http:// address silently
// downgrades the transport. Mutation proof: drop the SplitN handling and the
// "http://h" case resolves to scheme="https" (the cfg.Scheme), so the assertion
// scheme=="http" goes RED.
func TestResolveTarget(t *testing.T) {
	tests := []struct {
		name       string
		cfg        client.Config
		wantScheme string
		wantHost   string
	}{
		{"plain host keeps scheme", client.Config{Address: "recette.example", Scheme: "https"}, "https", "recette.example"},
		{"http scheme honored", client.Config{Address: "recette.example", Scheme: "http"}, "http", "recette.example"},
		{"scheme prefix overrides", client.Config{Address: "http://recette.example", Scheme: "https"}, "http", "recette.example"},
		{"empty scheme defaults https", client.Config{Address: "recette.example", Scheme: ""}, "https", "recette.example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			scheme, host := resolveTarget(&cfg)
			if scheme != tt.wantScheme || host != tt.wantHost {
				t.Fatalf("resolveTarget = %q://%q, want %q://%q", scheme, host, tt.wantScheme, tt.wantHost)
			}
		})
	}
}

// TestPreflightPrintsTarget pins that the resolved target is ALWAYS printed
// before firing (the #316 must-fix mitigation: the operator must see where the
// run is about to hit). Mutation proof: remove the Fprintf and the buffer no
// longer contains the target line, so this goes RED.
func TestPreflightPrintsTarget(t *testing.T) {
	var buf bytes.Buffer
	cfg := &client.Config{Address: "recette.example", Scheme: "https", ClientID: "id", SecretID: "sec", ApiSuffix: true}
	if err := preflight(cfg, &buf); err != nil {
		t.Fatalf("a https target with creds must pass preflight, got: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "target = https://recette.example") || !strings.Contains(got, "apiSuffix=true") {
		t.Fatalf("preflight must print the resolved target, got %q", got)
	}
}

// TestPreflightRefusesNonHTTPS pins that a cleartext scheme is refused so the
// PAT id/secret can never be sent in the open — both via CLOUDTEMPLE_HTTP_SCHEME
// and via an http:// prefix in the address. Mutation proof: remove the scheme
// check and both cases return nil, so the "expected an error" assertions go RED.
func TestPreflightRefusesNonHTTPS(t *testing.T) {
	cases := []client.Config{
		{Address: "recette.example", Scheme: "http", ClientID: "id", SecretID: "sec"},
		{Address: "http://recette.example", Scheme: "https", ClientID: "id", SecretID: "sec"},
	}
	for _, cfg := range cases {
		var buf bytes.Buffer
		c := cfg
		err := preflight(&c, &buf)
		if err == nil {
			t.Fatalf("preflight must refuse a non-https target %q", cfg.Address)
		}
		if !strings.Contains(err.Error(), "cleartext") {
			t.Fatalf("error should explain the cleartext risk, got: %v", err)
		}
		// The target is still printed even when refusing.
		if !strings.Contains(buf.String(), "target =") {
			t.Fatalf("preflight must print the target even when refusing, got %q", buf.String())
		}
	}
}

// TestPreflightRequiresCredentials pins fail-fast on missing creds (so the run
// does not fire empty-credential auth calls). Mutation proof: remove the creds
// check and a https target with empty creds returns nil, so this goes RED.
func TestPreflightRequiresCredentials(t *testing.T) {
	cases := []struct {
		name string
		cfg  client.Config
	}{
		{"no client id", client.Config{Address: "recette.example", Scheme: "https", SecretID: "sec"}},
		{"no secret id", client.Config{Address: "recette.example", Scheme: "https", ClientID: "id"}},
		{"neither", client.Config{Address: "recette.example", Scheme: "https"}},
		{"whitespace only", client.Config{Address: "recette.example", Scheme: "https", ClientID: "  ", SecretID: "\t"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := tt.cfg
			err := preflight(&cfg, &buf)
			if err == nil {
				t.Fatal("preflight must fail fast when credentials are missing")
			}
			if !strings.Contains(err.Error(), "credentials not set") {
				t.Fatalf("error should name the missing credentials, got: %v", err)
			}
		})
	}
}

// captureRun drives the real harness entrypoint (run takes *os.File) against
// temp files and returns the exit code plus captured stdout/stderr, so the
// preflight WIRING — not just preflight() in isolation — is exercised.
func captureRun(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	mk := func(name string) (*os.File, func() string) {
		f, err := os.CreateTemp(t.TempDir(), name)
		if err != nil {
			t.Fatalf("temp %s: %v", name, err)
		}
		return f, func() string {
			b, _ := os.ReadFile(f.Name())
			return string(b)
		}
	}
	outF, readOut := mk("stdout")
	errF, readErr := mk("stderr")
	code := run(args, outF, errF)
	_ = outF.Close()
	_ = errF.Close()
	return code, readOut(), readErr()
}

// TestRunListBypassesPreflight pins that -list exits 0 WITHOUT reaching the
// preflight (no resolved target printed, no client built, no network). Mutation
// proof: move the preflight ahead of the -list early return and -list then
// prints a "target =" line, so the "must not reach preflight" assertion goes RED.
func TestRunListBypassesPreflight(t *testing.T) {
	code, out, errOut := captureRun(t, "-list")
	if code != 0 {
		t.Fatalf("-list must exit 0, got %d (stderr=%q)", code, errOut)
	}
	if !strings.Contains(out, "Available cycles") {
		t.Fatalf("-list must list cycles, got %q", out)
	}
	if strings.Contains(errOut, "target =") {
		t.Fatalf("-list must not reach the preflight, but a target was printed: %q", errOut)
	}
}

// TestRunMissingCredsFailsFastNoNetwork pins the end-to-end wiring: a real cycle
// selection with missing credentials PRINTS the resolved target, then fails fast
// with exit code 2 (config error) BEFORE the client is built or any request is
// sent. The target is forced to a .example host so the test never depends on the
// production-looking default and never reaches the network (preflight returns
// before NewClient). Mutation proof: remove the preflight call in run() and a
// missing-creds invocation no longer exits 2 nor prints "credentials not set".
func TestRunMissingCredsFailsFastNoNetwork(t *testing.T) {
	t.Setenv(client.HTTPAddrEnvName, "recette.example")
	t.Setenv(client.HTTPSchemeEnvName, "https")
	t.Setenv(client.HTTPClientIDEnvName, "")
	t.Setenv(client.HTTPClientSecretEnvName, "")

	code, _, errOut := captureRun(t, "-cycles", "readonly")
	if code != 2 {
		t.Fatalf("missing creds must exit 2 (config error), got %d (stderr=%q)", code, errOut)
	}
	if !strings.Contains(errOut, "target = https://recette.example") {
		t.Fatalf("run must print the resolved target before failing, got %q", errOut)
	}
	if !strings.Contains(errOut, "credentials not set") {
		t.Fatalf("run must fail fast on missing creds, got %q", errOut)
	}
}
