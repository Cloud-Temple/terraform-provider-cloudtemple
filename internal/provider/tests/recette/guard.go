// Package recette hosts the LIVE acceptance ("recette") harness for issue #300,
// Phase 1: PAT + object storage on a TRULY EMPTY, dedicated recette tenant.
//
// This package is intentionally ISOLATED from internal/provider/tests so it can
// own its own TestMain: the existing tests TestMain performs destructive cleanup
// unconditionally and is unsafe to reuse for a live recette run (see #300 brief).
//
// SAFETY MODEL (non-negotiable, SecNumCloud posture):
//   - The harness can only ever touch the allowlisted recette tenant. The
//     allowlist is the runtime value of the env var CLOUDTEMPLE_RECETTE_TENANT_ID;
//     it is NEVER committed and NEVER printed.
//   - The guard is fail-closed and un-skippable: the live/sweep path runs the
//     auth + tenant assertion BEFORE anything mutating, and aborts fatally on
//     any mismatch.
//   - Guard errors are redacted: they never print the expected or the actual
//     tenant UUID.
//   - With TF_ACC unset and no -sweep flag, the guard is skipped and the pure
//     unit tests run WITHOUT credentials and WITHOUT any network call.
package recette

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// recetteTenantEnvName is the REQUIRED allowlist env var. Its runtime value is
// the recette tenant's JWT scope.id (a UUID). It is never committed; the guard
// reads it only at runtime and never prints it.
const recetteTenantEnvName = "CLOUDTEMPLE_RECETTE_TENANT_ID"

// errRecetteTenantUnset is returned when the allowlist env var is missing or
// empty in live/sweep mode. The harness must abort and mutate nothing.
var errRecetteTenantUnset = fmt.Errorf(
	"%s must be set to the recette tenant id to run the live recette harness; refusing to fall back to the current tenant",
	recetteTenantEnvName,
)

// errRecetteTenantMismatch is returned when the authenticated tenant does not
// match the allowlist. It is deliberately GENERIC: it never embeds the expected
// or the actual tenant UUID, so a failing run cannot leak either value.
var errRecetteTenantMismatch = fmt.Errorf(
	"authenticated tenant does not match %s; aborting before any mutation",
	recetteTenantEnvName,
)

// assertRecetteTenant is the PURE core of the guard. It compares the allowlisted
// tenant id (expected) with the authenticated tenant id (actual) and returns a
// redacted error on any failure. It performs NO I/O so it can be table-tested
// without credentials or a network call.
//
// Fail-closed semantics:
//   - empty expected  -> errRecetteTenantUnset (never trust an empty allowlist);
//   - empty actual    -> errRecetteTenantMismatch (an unauthenticated/blank tenant
//     can never satisfy the allowlist);
//   - expected != actual -> errRecetteTenantMismatch;
//   - expected == actual (both non-empty) -> nil.
//
// Neither returned error embeds either UUID, so the caller can print them
// verbatim without leaking a tenant id.
func assertRecetteTenant(expected, actual string) error {
	if expected == "" {
		return errRecetteTenantUnset
	}
	if actual == "" || actual != expected {
		return errRecetteTenantMismatch
	}
	return nil
}

// newRecetteClient builds a client from the standard CLOUDTEMPLE_* credentials
// (CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID / CLOUDTEMPLE_HTTP_ADDR ...).
// It NEVER reads .env.recette: credentials are injected into the process
// environment by the caller (CI environment secrets or a sourced, gitignored
// .env.recette), never loaded from a committed file.
func newRecetteClient() (*client.Client, error) {
	config := client.DefaultConfig()
	if config.ClientID == "" || config.SecretID == "" {
		return nil, fmt.Errorf(
			"%s and %s must be set to authenticate the recette harness",
			client.HTTPClientIDEnvName, client.HTTPClientSecretEnvName,
		)
	}
	return client.NewClient(config)
}

// guardLiveTenant performs the LIVE guard: read the allowlist env var,
// authenticate, resolve the authenticated tenant id, and assert it matches.
// It returns a redacted error on any failure and is the single un-skippable
// gate the live/sweep TestMain path runs before anything mutating.
//
// It NEVER prints any tenant UUID, client id, or secret: only the env-var NAME
// appears in its messages.
func guardLiveTenant(ctx context.Context) error {
	expected := os.Getenv(recetteTenantEnvName)
	if expected == "" {
		// Fail closed before even touching the network.
		return errRecetteTenantUnset
	}

	c, err := newRecetteClient()
	if err != nil {
		// Do not wrap with credential values; client errors are already
		// credential-free.
		return fmt.Errorf("recette guard: failed to build client: %w", err)
	}

	token, err := c.Token(ctx)
	if err != nil {
		return fmt.Errorf("recette guard: failed to authenticate: %w", err)
	}

	if err := assertRecetteTenant(expected, token.TenantID()); err != nil {
		return err
	}
	return nil
}

// abortRedacted prints a redacted message to stderr and exits non-zero. It is
// the only exit path of the live guard and is careful to print neither UUID nor
// credential. The errors it receives are already redacted by construction.
func abortRedacted(err error) {
	fmt.Fprintf(os.Stderr, "recette harness aborted: %s\n", err.Error())
	os.Exit(1)
}

// isRedactedGuardError reports whether err is one of the guard's redacted
// sentinels. It lets tests assert on the failure CLASS without ever comparing
// or printing a tenant id.
func isRedactedGuardError(err error) bool {
	return errors.Is(err, errRecetteTenantUnset) || errors.Is(err, errRecetteTenantMismatch)
}
