package recette

import (
	"os"
	"strings"
	"testing"
)

// TestRecetteLiveGuardOnly is a NON-MUTATING live pre-flight: in live mode
// (TF_ACC) it authenticates and asserts the tenant matches the allowlist, then
// returns. It creates and destroys NOTHING. The standalone HCL wrapper (run.sh)
// invokes it before `terraform apply` so the tenant guard runs once outside the
// SDK TestCase machinery too. Without TF_ACC it skips (no creds, no network).
func TestRecetteLiveGuardOnly(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("live tenant guard pre-flight skipped unless TF_ACC is set")
	}
	if err := guardLiveTenant(t.Context()); err != nil {
		// Redacted by construction: never prints a tenant id.
		t.Fatalf("recette tenant guard refused the run: %s", err)
	}
}

// TestRecetteGuardAssertTenant is the guard's proof. It table-tests the PURE
// comparison assertRecetteTenant with no TF_ACC and no network call.
//
// It is deliberately NON-COMPLACENT: it asserts the exact pass/fail boundary so
// the test goes RED if the comparison is inverted (e.g. == becomes !=) or if an
// empty input is silently accepted.
func TestRecetteGuardAssertTenant(t *testing.T) {
	const (
		recette = "11111111-1111-1111-1111-111111111111"
		other   = "22222222-2222-2222-2222-222222222222"
	)

	cases := []struct {
		name     string
		expected string
		actual   string
		// wantErr nil means the guard must let the run proceed.
		wantErr error
	}{
		{
			name:     "empty expected aborts (never trust an empty allowlist)",
			expected: "",
			actual:   recette,
			wantErr:  errRecetteTenantUnset,
		},
		{
			name:     "empty expected with empty actual still aborts as unset",
			expected: "",
			actual:   "",
			wantErr:  errRecetteTenantUnset,
		},
		{
			name:     "empty actual aborts (blank tenant cannot satisfy the allowlist)",
			expected: recette,
			actual:   "",
			wantErr:  errRecetteTenantMismatch,
		},
		{
			name:     "mismatch aborts",
			expected: recette,
			actual:   other,
			wantErr:  errRecetteTenantMismatch,
		},
		{
			name:     "exact match proceeds",
			expected: recette,
			actual:   recette,
			wantErr:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := assertRecetteTenant(tc.expected, tc.actual)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected the guard to PROCEED, got error: %v", err)
				}
				return
			}
			if err != tc.wantErr { //nolint:errorlint // sentinel identity is the contract here
				t.Fatalf("expected sentinel %v, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRecetteGuardErrorsAreRedacted proves the guard never leaks a tenant UUID.
// Both the sentinel errors and isRedactedGuardError must hold without embedding
// either id.
func TestRecetteGuardErrorsAreRedacted(t *testing.T) {
	const (
		recette = "11111111-1111-1111-1111-111111111111"
		other   = "22222222-2222-2222-2222-222222222222"
	)

	for _, actual := range []string{"", other} {
		err := assertRecetteTenant(recette, actual)
		if err == nil {
			t.Fatalf("expected an error for actual=%q", actual)
		}
		msg := err.Error()
		// The redacted message must mention neither the expected nor the
		// actual tenant id.
		if strings.Contains(msg, recette) {
			t.Fatalf("guard error leaked the expected tenant id: %q", msg)
		}
		if actual != "" && strings.Contains(msg, actual) {
			t.Fatalf("guard error leaked the actual tenant id: %q", msg)
		}
		if !strings.Contains(msg, recetteTenantEnvName) {
			t.Fatalf("guard error should name the allowlist env var, got: %q", msg)
		}
		if !isRedactedGuardError(err) {
			t.Fatalf("guard error not recognized as a redacted sentinel: %v", err)
		}
	}
}

// TestRecetteGuardUnsetIsRedacted proves the unset-allowlist abort is also
// redacted and recognized.
func TestRecetteGuardUnsetIsRedacted(t *testing.T) {
	err := assertRecetteTenant("", "anything")
	if err != errRecetteTenantUnset { //nolint:errorlint // sentinel identity is the contract here
		t.Fatalf("expected errRecetteTenantUnset, got %v", err)
	}
	if !isRedactedGuardError(err) {
		t.Fatalf("unset error not recognized as a redacted sentinel: %v", err)
	}
	if !strings.Contains(err.Error(), recetteTenantEnvName) {
		t.Fatalf("unset error should name the allowlist env var, got: %q", err.Error())
	}
}
