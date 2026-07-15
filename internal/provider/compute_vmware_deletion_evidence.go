package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// confirmVMwareDeviceLiveness reports whether a device id is still present in a
// strict, scoped listing. It NEVER concludes a deletion: even a nil per-id read
// (since #384 a definitive 404; a genuine 403 surfaces as an access-denied error)
// is not deletion evidence here, because the scoped listing is not provably
// complete tenant-wide. The caller never drops the resource on this signal (#281).
//
// Empty ids in the listing are skipped; matching is on the exact id.
func confirmVMwareDeviceLiveness(ctx context.Context, id string, listScoped func(ctx context.Context) ([]string, error)) (bool, error) {
	ids, err := listScoped(ctx)
	if err != nil {
		return false, err
	}
	for _, candidate := range ids {
		if candidate != "" && candidate == id {
			return true, nil
		}
	}
	return false, nil
}

// confirmVMwareDeviceOrKeep handles the nil-read branch of a VMware resource's
// Read. It ALWAYS returns an error diagnostic and NEVER touches the
// ResourceData, so it is structurally incapable of dropping the resource: the
// resource is kept in the state in every case, and the read never succeeds on
// an unreadable resource.
//
// A nil per-id read is handled conservatively (since #384 a definitive 404; a
// genuine 403 surfaces as an access-denied error before reaching here). The
// resource is never auto-removed; but it is also never reported as a SUCCESSFUL
// refresh, because the attributes could not be re-read and the state may be
// stale — silently succeeding would make Terraform treat unrefreshed,
// potentially drifted state as converged. So the read fails closed with an
// actionable diagnostic, mirroring the OpenIaaS "still listed -> refuse and
// error" behavior. The scoped listing only sharpens the diagnostic (still
// listed = likely transient/permission; absent = possibly deleted).
//
// kind is the human label of the resource (e.g. "virtual disk"); scopeLabel is
// the label of the scoping parent (e.g. "virtual machine"); scopeID is its id
// read from the state. An empty scopeID fails closed: an unscoped listing is
// not safe evidence, so there is nothing to confirm against.
func confirmVMwareDeviceOrKeep(ctx context.Context, id, kind, scopeLabel, scopeID string, listScoped func(ctx context.Context) ([]string, error)) diag.Diagnostics {
	if scopeID == "" {
		return diag.Errorf(
			"%s %s could not be read and its existence could not be confirmed because the %s id is missing from the state; the resource is kept in the state to avoid a wrong deletion. Resolve the read error, then refresh or re-import it.",
			kind, id, scopeLabel,
		)
	}
	present, err := confirmVMwareDeviceLiveness(ctx, id, listScoped)
	if err != nil {
		return diag.Errorf(
			"%s %s could not be read and its existence could not be confirmed (the listing failed); the resource is kept in the state to avoid a wrong deletion: %s",
			kind, id, err,
		)
	}
	if present {
		// Liveness confirmed but the per-id read failed: the resource still
		// exists (likely a transient error or an access restriction), so we
		// refuse to drop it. We also refuse to report a successful refresh,
		// because the attributes could not be re-read and the state may be
		// stale. Fail closed with an error; the next refresh re-reads it.
		return diag.Errorf(
			"%s %s could not be read but is still listed under %s %s; the resource is kept in the state (refusing to drop it on a likely transient error or access restriction). Its attributes could not be refreshed — retry once the read succeeds.",
			kind, id, scopeLabel, scopeID,
		)
	}
	return diag.Errorf(
		"%s %s is no longer returned by the API and is not listed under %s %s; its deletion could not be confirmed (it may have been deleted, detached or moved, or your access may have changed). The resource is kept in the state. If you removed it intentionally, run `terraform state rm` on it.",
		kind, id, scopeLabel, scopeID,
	)
}
