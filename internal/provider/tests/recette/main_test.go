package recette

import (
	"context"
	"flag"
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

// TestMain is the un-skippable safety core of the recette harness. On its own it
// MUTATES NOTHING: it only authenticates and asserts the tenant, then delegates.
//
//   - In live mode (TF_ACC) OR sweep mode (-sweep): it runs the tenant guard +
//     auth assertion FIRST and aborts fatally on any mismatch. This is a pure
//     auth + tenant assertion; it creates and destroys nothing.
//   - In the default path (no TF_ACC, no -sweep): the guard is skipped entirely;
//     the pure unit tests run WITHOUT credentials and without any network call.
//
// There is NO automatic start-of-run cleanup. A clean slate is an EXPLICIT,
// destructive step the operator runs on purpose via the -sweep flag (below);
// never a side effect of a live test run. Removing the auto-sweep keeps the
// principle of least surprise: invoking TestMain in live mode — as the run.sh
// guard pre-flight and TestRecetteLiveGuardOnly do — deletes nothing.
//
// In every case it DELEGATES to resource.TestMain(m): that helper runs the
// registered sweepers when -sweep is set, otherwise it calls m.Run(). We never
// call a bare m.Run() — that would silently break -sweep. The ONLY destructive
// path is the explicit -sweep, where resource.TestMain fires the registered
// sweepers (each re-asserting the tenant guard via sweepRecette before deleting).
func TestMain(m *testing.M) {
	// Parse flags so -sweep is observable before we branch. resource.TestMain
	// calls flag.Parse() again, which is idempotent.
	flag.Parse()

	if liveEnabled() || sweepEnabled() {
		ctx := context.Background()

		// Fail-closed un-skippable gate: prove the credentials point at the
		// allowlisted recette tenant BEFORE anything mutating runs. This is the
		// only thing TestMain does in live/sweep mode; it mutates nothing.
		if err := guardLiveTenant(ctx); err != nil {
			abortRedacted(err)
		}
	}

	// Delegates to the SDK: runs sweepers under -sweep, otherwise m.Run().
	// resource.TestMain calls os.Exit itself.
	resource.TestMain(m)
}
