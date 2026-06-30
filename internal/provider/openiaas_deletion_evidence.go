package provider

// missingDeviceVerdict is the decision for a resource whose per-id read
// returned nil: since #384 a per-id 403 surfaces as an access-denied error and
// only a definitive 404 maps to nil, and a deletion is still accepted only
// under strict listing evidence (#275 doctrine, FF-5).
type missingDeviceVerdict int

const (
	// deviceDeletionConfirmed: absent from both the scoped and the
	// tenant-wide strict listings — the deletion is proven, the state
	// entry can be dropped.
	deviceDeletionConfirmed missingDeviceVerdict = iota
	// deviceStillInScope: the strict scoped listing still contains the id
	// while the per-id read returned nil — access restriction or API
	// inconsistency, fail closed.
	deviceStillInScope
	// deviceExistsOutOfScope: absent from the scoped listing but present
	// tenant-wide — the device was detached or moved platform-side, which
	// is drift, never a deletion. Fail closed with an actionable message.
	deviceExistsOutOfScope
)

// classifyMissingDevice renders the verdict from the two strict listings.
func classifyMissingDevice(id string, scopedIDs map[string]bool, tenantIDs map[string]bool) missingDeviceVerdict {
	if scopedIDs[id] {
		return deviceStillInScope
	}
	if tenantIDs[id] {
		return deviceExistsOutOfScope
	}
	return deviceDeletionConfirmed
}
