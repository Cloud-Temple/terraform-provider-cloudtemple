package recette

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	providerpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// providerFactories instantiate the production provider for the recette
// TestCases. It mirrors internal/provider/tests/provider_test.go (api_suffix
// disabled in dev) WITHOUT importing that package, so the unsafe tests TestMain
// is never linked into this binary.
var providerFactories = map[string]func() (*schema.Provider, error){
	"cloudtemple": func() (*schema.Provider, error) {
		p := providerpkg.New("dev")()
		p.Schema["api_suffix"].Default = false
		return p, nil
	},
}

// sweepEnabled reports whether the SDK -sweep flag was passed. The flag is owned
// by the SDK resource package; we read it from the global flag set after
// flag.Parse(). Sweep mode is destructive, so it must also pass the tenant
// guard before anything is deleted.
func sweepEnabled() bool {
	f := flag.Lookup("sweep")
	return f != nil && f.Value.String() != ""
}

// liveEnabled reports whether the live acceptance path is active (TF_ACC set).
func liveEnabled() bool {
	return os.Getenv(resource.EnvTfAcc) != ""
}

// TestMain is the un-skippable safety core of the recette harness.
//
//   - In live mode (TF_ACC) OR sweep mode (-sweep): it runs the tenant guard +
//     auth assertion FIRST and aborts fatally (mutating nothing) on any mismatch,
//     then runs the guarded start-of-run cleanup for a clean slate.
//   - In the default path (no TF_ACC, no -sweep): the guard is skipped entirely;
//     the pure unit tests run WITHOUT credentials and without any network call.
//
// In every case it DELEGATES to resource.TestMain(m): that helper runs the
// registered sweepers when -sweep is set, otherwise it calls m.Run(). We never
// call a bare m.Run() — that would silently break -sweep.
func TestMain(m *testing.M) {
	// Parse flags so -sweep is observable before we branch. resource.TestMain
	// calls flag.Parse() again, which is idempotent.
	flag.Parse()

	if liveEnabled() || sweepEnabled() {
		ctx := context.Background()

		// Fail-closed un-skippable gate: prove the credentials point at the
		// allowlisted recette tenant BEFORE anything mutating runs.
		if err := guardLiveTenant(ctx); err != nil {
			abortRedacted(err)
		}

		// Belt-and-braces start-of-run cleanup (clean slate), itself guarded.
		// This is the explicit cleanup hook described in D3(b): it is NOT
		// wired through AddTestSweepers (which only fires under -sweep), so it
		// runs at the start of any live run too. It is intentionally skipped
		// under -sweep, where resource.TestMain runs the registered sweepers
		// (which share the same body via sweepRecette).
		if liveEnabled() && !sweepEnabled() {
			if err := sweepRecette(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "recette start-of-run cleanup failed: %s\n", err.Error())
				os.Exit(1)
			}
		}
	}

	// Delegates to the SDK: runs sweepers under -sweep, otherwise m.Run().
	// resource.TestMain calls os.Exit itself.
	resource.TestMain(m)
}
