package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// confirmVMwareDeviceLiveness reports whether a device id is still present in a
// strict, scoped listing. It NEVER concludes a deletion: a nil per-id read on a
// VMware resource is ambiguous (the client maps HTTP 403 to nil), and the
// scoped listing is not provably complete tenant-wide, so absence is not
// deletion evidence. The caller keeps the resource in the state in every case
// except a confirmed liveness (#281).
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
// Read. It returns diagnostics but NEVER touches the ResourceData, so it is
// structurally incapable of dropping the resource from the state: the resource
// is kept in every inconclusive case. It returns nil diagnostics only when the
// device's liveness is confirmed by the scoped listing (the per-id read was a
// transient or permission blip), in which case the caller keeps the existing
// state untouched.
//
// kind is the human label of the resource (e.g. "virtual disk"); scopeLabel is
// the label of the scoping parent (e.g. "virtual machine"); scopeID is its id
// read from the state. An empty scopeID fails closed: an unscoped listing is
// not safe evidence, so there is nothing to confirm against.
func confirmVMwareDeviceOrKeep(ctx context.Context, id, kind, scopeLabel, scopeID string, listScoped func(ctx context.Context) ([]string, error)) diag.Diagnostics {
	if scopeID == "" {
		return diag.Errorf(
			"%s %s could not be read and its existence could not be confirmed because the %s id is missing from the state; the resource is kept in the state to avoid a wrong deletion. Refresh or re-import it once it is readable again.",
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
	if !present {
		return diag.Errorf(
			"%s %s is no longer returned by the API and is not listed under %s %s; its deletion could not be confirmed (it may have been deleted, detached or moved, or your access may have changed). The resource is kept in the state. If you removed it intentionally, run `terraform state rm` on it.",
			kind, id, scopeLabel, scopeID,
		)
	}
	// Liveness confirmed: the per-id read was a transient or permission blip.
	// Keep the existing state untouched — we do not repopulate from the scoped
	// listing (the next successful per-id read does that), which also avoids
	// trusting the listing payload for this resource's attributes.
	return nil
}
