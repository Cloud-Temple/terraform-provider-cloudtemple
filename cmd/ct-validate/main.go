// Command ct-validate is a parametrizable endpoint validation & resilience
// harness for the Cloud Temple provider. It exercises the provider's endpoints
// THROUGH the internal/client library (not through Terraform), runs realistic
// business cycles, and reports WHERE IT SQUEAKS: per cycle/endpoint success
// rate, latency p50/p95, and a failure-category histogram.
//
// Safety is the reason it exists (incident 2026-06-15: a high-frequency write
// loop amplified an API outage and orphaned resources). Therefore:
//   - -write defaults to false: with default flags the tool only READS;
//   - a circuit breaker is always active and stops launching new work the
//     moment the API shows distress, then tears down what was created;
//   - default concurrency is low (2): high concurrency on the SHARED recette
//     API is dangerous and must be a deliberate choice;
//   - there are no infinite retries anywhere; teardown is one attempt plus a
//     small bounded transient-retry.
//
// -list and -help work WITHOUT a network and WITHOUT constructing the client.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// buildRegistry assembles every available cycle. Keeping it in one place lets
// -list, the selector, and the engine share the exact same set.
func buildRegistry() *Registry {
	return NewRegistry(
		readonlyCycle{},
		backupCycle{},
		computeOpenIaaSCycle{},
		vpcCycle{},
		objectStorageCycle{},
		iamPATCycle{},
	)
}

type flags struct {
	runs          int
	concurrency   int
	cycles        string
	write         bool
	timeout       time.Duration
	abortFailRate float64
	abortConsec   int
	abortWindow   int
	jsonOut       bool
	list          bool
	apiSuffix     bool
}

func parseFlags(args []string, out *os.File) (*flags, error) {
	fs := flag.NewFlagSet("ct-validate", flag.ContinueOnError)
	fs.SetOutput(out)
	f := &flags{}
	fs.IntVar(&f.runs, "runs", 1, "iterations of each selected cycle")
	fs.IntVar(&f.concurrency, "concurrency", 2, "worker pool size (HIGH VALUES ARE DANGEROUS on the shared API)")
	fs.StringVar(&f.cycles, "cycles", "readonly", "comma-separated cycles to run, or \"all\"")
	fs.BoolVar(&f.write, "write", false, "enable WRITE cycles (skipped by default; only reads run without this)")
	fs.DurationVar(&f.timeout, "timeout", 30*time.Minute, "global timeout for the whole run")
	fs.Float64Var(&f.abortFailRate, "abort-failure-rate", 0.30, "trip the breaker when the failure rate over the window reaches this (0..1)")
	fs.IntVar(&f.abortConsec, "abort-consecutive", 5, "trip the breaker after this many consecutive failures")
	fs.IntVar(&f.abortWindow, "abort-window", 20, "rolling window size for the failure-rate rule")
	fs.BoolVar(&f.jsonOut, "json", false, "emit the report as JSON")
	fs.BoolVar(&f.list, "list", false, "list available cycles and exit (no network)")
	fs.BoolVar(&f.apiSuffix, "api-suffix", true, "prefix request paths with /api (client ApiSuffix)")

	fs.Usage = func() {
		fmt.Fprintf(out, "ct-validate — endpoint validation & resilience harness (reads through internal/client)\n\n")
		fmt.Fprintf(out, "USAGE:\n  ct-validate [flags]\n\n")
		fmt.Fprintf(out, "SAFETY:\n")
		fmt.Fprintf(out, "  -write defaults false (reads only). The circuit breaker is always on and stops\n")
		fmt.Fprintf(out, "  launching work the moment the API squeaks, then tears down. Keep concurrency low\n")
		fmt.Fprintf(out, "  on the shared recette API.\n\n")
		fmt.Fprintf(out, "CREDENTIALS (env, only needed to actually run cycles — NOT for -list/-help):\n")
		fmt.Fprintf(out, "  %s, %s, %s\n\n",
			client.HTTPClientIDEnvName, client.HTTPClientSecretEnvName, client.HTTPAddrEnvName)
		fmt.Fprintf(out, "FLAGS:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := f.validate(); err != nil {
		fs.Usage()
		return nil, err
	}
	return f, nil
}

func (f *flags) validate() error {
	if f.runs < 1 {
		return fmt.Errorf("-runs must be >= 1 (got %d)", f.runs)
	}
	if f.concurrency < 1 {
		return fmt.Errorf("-concurrency must be >= 1 (got %d)", f.concurrency)
	}
	if f.timeout <= 0 {
		return fmt.Errorf("-timeout must be > 0 (got %s)", f.timeout)
	}
	if f.abortFailRate <= 0 || f.abortFailRate > 1 {
		return fmt.Errorf("-abort-failure-rate must be in (0,1] (got %g)", f.abortFailRate)
	}
	if f.abortConsec < 1 {
		return fmt.Errorf("-abort-consecutive must be >= 1 (got %d)", f.abortConsec)
	}
	if f.abortWindow < 1 {
		return fmt.Errorf("-abort-window must be >= 1 (got %d)", f.abortWindow)
	}
	return nil
}

// resolveTarget computes the scheme and host the client will ACTUALLY use,
// mirroring NewClient's handling of a "scheme://" prefix embedded in the
// address (internal/client/api.go). Precondition: cfg has already been resolved
// by client.DefaultConfig() (as run() does), so cfg.Scheme/cfg.Address already
// reflect the environment; under that precondition it prints/checks the exact
// same target the requests will hit. An empty scheme defaults to https, matching
// DefaultConfig.
func resolveTarget(cfg *client.Config) (scheme, host string) {
	scheme, host = cfg.Scheme, cfg.Address
	if parts := strings.SplitN(cfg.Address, "://", 2); len(parts) == 2 {
		scheme, host = parts[0], parts[1]
	}
	if scheme == "" {
		scheme = "https"
	}
	return scheme, host
}

// preflight enforces the live-run safety guards the #316 audit flagged before
// any request is issued against the SHARED recette API:
//   - it PRINTS the resolved target (scheme://host) so the operator can confirm
//     it is the intended recette host — the binary otherwise never shows where
//     it is about to fire, and the default address looks production-like;
//   - it REFUSES a non-HTTPS scheme: the credential exchange carries the PAT
//     id/secret and must never travel in cleartext;
//   - it FAILS FAST when credentials are missing, instead of firing
//     empty-credential auth calls.
//
// The target line goes to w (stderr in run()); stdout stays reserved for the
// report / JSON.
func preflight(cfg *client.Config, w io.Writer) error {
	scheme, host := resolveTarget(cfg)
	fmt.Fprintf(w, "ct-validate: target = %s://%s (apiSuffix=%t) — confirm this is the intended RECETTE host\n",
		scheme, host, cfg.ApiSuffix)

	if scheme != "https" {
		return fmt.Errorf("refusing to run over %q: the credential exchange would travel in cleartext; "+
			"use https (unset %s or set it to https, and drop any \"http://\" prefix in %s)",
			scheme, client.HTTPSchemeEnvName, client.HTTPAddrEnvName)
	}
	if strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.SecretID) == "" {
		return fmt.Errorf("credentials not set: export %s and %s before running cycles",
			client.HTTPClientIDEnvName, client.HTTPClientSecretEnvName)
	}
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entrypoint. It returns the process exit code:
//
//	0 = success (no failures), 1 = failures present or error, 2 = usage error.
func run(args []string, stdout, stderr *os.File) int {
	f, err := parseFlags(args, stderr)
	if err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(stderr, "ct-validate: %v\n", err)
		return 2
	}

	reg := buildRegistry()

	if f.list {
		fmt.Fprintln(stdout, "Available cycles:")
		for _, c := range reg.All() {
			fmt.Fprintf(stdout, "  %-18s [%s]\n", c.Name(), c.Kind())
		}
		fmt.Fprintln(stdout, "\nWrite cycles run only with -write. Default selection is \"readonly\".")
		return 0
	}

	selected, gated, err := reg.Select(f.cycles, f.write)
	if err != nil {
		fmt.Fprintf(stderr, "ct-validate: %v\n", err)
		return 2
	}
	if len(selected) == 0 {
		fmt.Fprintf(stderr, "ct-validate: nothing to run (selection %q resolved to no runnable cycles; "+
			"gated write cycles: %v — pass -write to enable them)\n", f.cycles, gated)
		return 2
	}

	// Build the client only now: -list/-help never reach here, so they need no
	// network and no credentials.
	cfg := client.DefaultConfig()
	cfg.ApiSuffix = f.apiSuffix

	// Safety preflight: print the resolved target, refuse cleartext, and require
	// credentials BEFORE building the client or firing any request (#316 audit).
	if err := preflight(cfg, stderr); err != nil {
		fmt.Fprintf(stderr, "ct-validate: %v\n", err)
		return 2
	}

	c, err := client.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "ct-validate: building client: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	breaker := NewBreaker(f.abortConsec, f.abortFailRate, f.abortWindow)
	rec := NewRecorder()
	cleanup := NewCleanup()
	engine := NewEngine(EngineConfig{Runs: f.runs, Concurrency: f.concurrency}, breaker, rec, cleanup)

	res := engine.Run(ctx, c, selected, gated)

	if f.jsonOut {
		if err := PrintJSON(stdout, res); err != nil {
			fmt.Fprintf(stderr, "ct-validate: encoding JSON: %v\n", err)
			return 1
		}
	} else {
		PrintText(stdout, res)
	}

	// CI/automation friendly: non-zero if any cycle had failures, a teardown
	// failed (possible orphan), or the breaker tripped.
	if HasFailure(res.Stats) || len(res.TeardownFailed) > 0 || res.Tripped {
		return 1
	}
	return 0
}
