package client

import "fmt"

// Capacity field rename compatibility (issue #524).
//
// The VM Instances API renamed every capacity field from the `*Gb`/`*Mb`
// spelling to `*Gib`/`*Mib`, to match the binary units the platform has always
// used. The rename is a PURE KEY RENAME: upstream every conversion is binary
// (1024**3, 1024), so the deprecated name is an EXACT ALIAS of the current one,
// never a converted quantity. The API re-emits both spellings during a
// compatibility window that closes on 2026-12-09.
//
// The Public Cloud VM structs carry no json tags and rely on encoding/json
// case-insensitive matching, so a field named after the deprecated spelling
// would silently decode to 0 once the API stops sending it — a zero the
// provider cannot distinguish from a genuine value, written straight into the
// Terraform state. The resolvers below make that impossible: reads accept both
// spellings, prefer the current one, and fail closed rather than yield a zero
// the API never sent.
//
// REMOVAL: issue #525 tracks dropping the deprecated branch once the window has
// closed AND the absence has been verified live (an announced removal date is
// not evidence of removal).

// capacityMismatchError reports the two spellings of a renamed capacity field
// disagreeing. The deprecated name is contractually an exact alias, so a
// divergence means something between the provider and the API rewrote the value
// — a unit conversion applied by a proxy being the dangerous case. Neither
// value can be trusted, so the read fails instead of picking one.
func capacityMismatchError(subject, currentKey, deprecatedKey string, current, deprecated any) error {
	return fmt.Errorf(
		"the VM Instances API returned conflicting capacity values for %s: %q=%v but the deprecated %q=%v. "+
			"The deprecated field must carry the same value as the current one, so the provider cannot "+
			"determine which is authoritative and refuses to guess. Check whether a proxy or an intermediate "+
			"API version is rewriting these fields",
		subject, currentKey, current, deprecatedKey, deprecated,
	)
}

// capacityAbsentError reports a required capacity field the API sent under
// neither spelling.
func capacityAbsentError(subject, currentKey, deprecatedKey string) error {
	return fmt.Errorf(
		"the VM Instances API returned neither %q nor the deprecated %q for %s, so the provider cannot "+
			"determine its capacity. This requires a VM Instances API version that serves one of these "+
			"fields; the provider refuses to record an unknown capacity as 0",
		currentKey, deprecatedKey, subject,
	)
}

// resolveRenamedCapacity resolves a REQUIRED renamed capacity field. Either
// spelling alone is accepted, which is what lets one provider build serve an API
// that has done the rename and one that has not. When BOTH are present they must
// agree — so the "current wins" branch below is a formality, never an
// arbitration: a divergence is rejected before it can be reached. An absence
// under both spellings is an error, never a zero: the API contract marks these
// fields required, so their absence is a broken contract, not a value.
func resolveRenamedCapacity(current, deprecated *int, currentKey, deprecatedKey, subject string) (int, error) {
	switch {
	case current != nil && deprecated != nil:
		if *current != *deprecated {
			return 0, capacityMismatchError(subject, currentKey, deprecatedKey, *current, *deprecated)
		}
		return *current, nil
	case current != nil:
		return *current, nil
	case deprecated != nil:
		return *deprecated, nil
	default:
		return 0, capacityAbsentError(subject, currentKey, deprecatedKey)
	}
}

// resolveOptionalRenamedCapacity resolves a renamed capacity field the API
// contract declares with `default: 0` rather than as required (the quota usage
// counters). Absence is legitimate there and yields 0; a divergence between the
// two spellings is still an error, for the same reason as above.
func resolveOptionalRenamedCapacity(current, deprecated *int, currentKey, deprecatedKey, subject string) (int, error) {
	if current == nil && deprecated == nil {
		return 0, nil
	}
	return resolveRenamedCapacity(current, deprecated, currentKey, deprecatedKey, subject)
}

// resolveRenamedCapacityList is resolveRenamedCapacity for a list-valued
// capacity field (the image disk sizes). Presence is what distinguishes the two
// spellings — an empty list the API genuinely sent is preserved as such, and
// only an absence under both spellings is an error.
func resolveRenamedCapacityList(current, deprecated *[]int, currentKey, deprecatedKey, subject string) ([]int, error) {
	switch {
	case current != nil && deprecated != nil:
		if !equalIntSlices(*current, *deprecated) {
			return nil, capacityMismatchError(subject, currentKey, deprecatedKey, *current, *deprecated)
		}
		return *current, nil
	case current != nil:
		return *current, nil
	case deprecated != nil:
		return *deprecated, nil
	default:
		return nil, capacityAbsentError(subject, currentKey, deprecatedKey)
	}
}

func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// quotedOrUnidentified renders an object identifier for a diagnostic. An empty
// id (a malformed payload is exactly the situation these errors report) must not
// produce a dangling `disk ""`, so it degrades to an explicit marker.
func quotedOrUnidentified(id string) string {
	if id == "" {
		return "(no id in the payload)"
	}
	return `"` + id + `"`
}
