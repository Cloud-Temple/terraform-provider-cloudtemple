package helpers

import (
	"reflect"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// This file holds a reusable, table-driven harness that asserts the GENERIC
// state-facing invariants every Flatten* helper must satisfy. It is the
// content-blind counterpart of the structural walker in the provider package
// (TestDatasourceFlattenOutputsFitTheirSchemas): the walker feeds a
// fully-populated struct and checks the output FITS the datasource schema;
// this harness feeds an EMPTY (zero-valued) struct and checks the output is
// SAFE to hand to the Terraform state writer. The two are complementary and
// must not be merged.
//
// Invariants asserted here, for each registered helper:
//
//   (i)   No panic when the helper is called on a fully zero-valued struct
//         pointer. This is the realistic Read contract: every Read decodes a
//         non-nil struct (possibly with empty fields) before flattening, so a
//         crash on an empty-but-present object is a real datasource breakage
//         (a list with one bare element, an object with no sub-resources).
//
//         NOTE on nil pointers: the helpers do NOT guard a nil receiver and
//         WOULD panic on one (e.g. FlattenRole(nil) dereferences role.ID).
//         The Read paths never pass nil, so this harness does not assert
//         nil-pointer safety: doing so would require a defensive guard in
//         ~50 production helpers for an input that never occurs. We pin the
//         realistic contract (zero-valued struct) instead, and leave the nil
//         case as a documented non-goal. See the PR body.
//
//   (ii)  The returned map is non-nil. A nil map handed to d.Set per key is a
//         silent no-op that drops the whole object from the state.
//
//   (iii) Every slice-typed value the helper emits from an EMPTY source is a
//         non-nil empty slice ([]interface{}{} / []map[string]interface{}{}),
//         never a typed nil — UNLESS the (helper,key) pair is in the explicit
//         knownNilSliceGaps registry below. Emitting [] pins the "present but
//         empty" intent rather than relying on the SDK silently normalizing a
//         nil slice to []. This invariant is a code-quality / intent contract,
//         not a crash class: a typed-nil slice under a TypeList is currently
//         tolerated by the SDK writer (it normalizes nil and [] to the same
//         stored []interface{}{}). The registry makes the helpers that still
//         pass a raw nil-able field through VISIBLE and tracked, exactly like
//         the walker's datasourceKnownGaps map, instead of either silently
//         accepting them (complacent) or touching production code to fix them
//         (out of scope for a test-only PR). Each gap is a debt to close in a
//         later batch by making the helper emit [] explicitly.

// zeroFlattenCase describes one Flatten* helper to run the generic invariants
// against. build calls the helper on a freshly-allocated zero-valued struct of
// the helper's input type. Using a typed closure keeps the registry free of
// reflection while still exercising the real helper.
type zeroFlattenCase struct {
	name  string
	build func() map[string]interface{}
}

// zeroOf adapts a Flatten helper of the canonical signature
// func(*T) map[string]interface{} into a builder that invokes it on a
// zero-valued *T. The struct is present (non-nil) but every field is its zero
// value, mirroring an API object decoded with no populated sub-resources.
func zeroOf[T any](flatten func(*T) map[string]interface{}) func() map[string]interface{} {
	return func() map[string]interface{} {
		var v T
		return flatten(&v)
	}
}

// genericFlattenCases registers the Flatten* helpers that cleanly fit the
// generic contract (single *T argument, no extra parameters, no required
// non-zero invariants on the input). Helpers with extra arguments
// (FlattenOpenIaaSVirtualDisk, the *Data sub-flatteners) are exercised by
// their dedicated tests instead. New helpers should be added here as they are
// written; the count is asserted below so the registry cannot silently rot.
func genericFlattenCases() []zeroFlattenCase {
	return []zeroFlattenCase{
		// --- VMware compute (vCenter) -------------------------------------
		{"FlattenContentLibrary", zeroOf(FlattenContentLibrary)},
		{"FlattenContentLibraryItem", zeroOf(FlattenContentLibraryItem)},
		{"FlattenDatastore", zeroOf(FlattenDatastore)},
		{"FlattenDatastoreCluster", zeroOf(FlattenDatastoreCluster)},
		{"FlattenFolder", zeroOf(FlattenFolder)},
		{"FlattenGuestOperatingSystem", zeroOf(FlattenGuestOperatingSystem)},
		{"FlattenHost", zeroOf(FlattenHost)},
		{"FlattenHostCluster", zeroOf(FlattenHostCluster)},
		{"FlattenNetwork", zeroOf(FlattenNetwork)},
		{"FlattenNetworkAdapter", zeroOf(FlattenNetworkAdapter)},
		{"FlattenResourcePool", zeroOf(FlattenResourcePool)},
		{"FlattenSnapshot", zeroOf(FlattenSnapshot)},
		{"FlattenVirtualController", zeroOf(FlattenVirtualController)},
		{"FlattenVirtualDatacenter", zeroOf(FlattenVirtualDatacenter)},
		{"FlattenVirtualDisk", zeroOf(FlattenVirtualDisk)},
		{"FlattenVirtualMachine", zeroOf(FlattenVirtualMachine)},
		{"FlattenVirtualSwitch", zeroOf(FlattenVirtualSwitch)},
		{"FlattenWorker", zeroOf(FlattenWorker)},

		// --- Backup (SPP) -------------------------------------------------
		{"FlattenBackupJob", zeroOf(FlattenBackupJob)},
		{"FlattenBackupJobSession", zeroOf(FlattenBackupJobSession)},
		{"FlattenBackupSite", zeroOf(FlattenBackupSite)},
		{"FlattenBackupSLAPolicy", zeroOf(FlattenBackupSLAPolicy)},
		{"FlattenBackupSPPServer", zeroOf(FlattenBackupSPPServer)},
		{"FlattenBackupStorage", zeroOf(FlattenBackupStorage)},
		{"FlattenBackupVCenter", zeroOf(FlattenBackupVCenter)},
		{"FlattenBackupMetricsCoverage", zeroOf(FlattenBackupMetricsCoverage)},
		{"FlattenBackupMetricsHistory", zeroOf(FlattenBackupMetricsHistory)},
		{"FlattenBackupMetricsPlatform", zeroOf(FlattenBackupMetricsPlatform)},
		{"FlattenBackupMetricsPlatformCPU", zeroOf(FlattenBackupMetricsPlatformCPU)},
		{"FlattenBackupMetricsPolicy", zeroOf(FlattenBackupMetricsPolicy)},
		{"FlattenBackupMetricsVirtualMachines", zeroOf(FlattenBackupMetricsVirtualMachines)},

		// --- Backup (OpenIaaS) --------------------------------------------
		{"FlattenBackupOpenIaasBackup", zeroOf(FlattenBackupOpenIaasBackup)},
		{"FlattenBackupOpenIaasPolicy", zeroOf(FlattenBackupOpenIaasPolicy)},

		// --- Compute (OpenIaaS) -------------------------------------------
		{"FlattenOpenIaaSHost", zeroOf(FlattenOpenIaaSHost)},
		{"FlattenOpenIaaSMachineManager", zeroOf(FlattenOpenIaaSMachineManager)},
		{"FlattenOpenIaaSNetwork", zeroOf(FlattenOpenIaaSNetwork)},
		{"FlattenOpenIaaSNetworkAdapter", zeroOf(FlattenOpenIaaSNetworkAdapter)},
		{"FlattenOpenIaaSPool", zeroOf(FlattenOpenIaaSPool)},
		{"FlattenOpenIaaSReplicationPolicy", zeroOf(FlattenOpenIaaSReplicationPolicy)},
		{"FlattenOpenIaaSSnapshot", zeroOf(FlattenOpenIaaSSnapshot)},
		{"FlattenOpenIaaSStorageRepository", zeroOf(FlattenOpenIaaSStorageRepository)},
		{"FlattenOpenIaaSTemplate", zeroOf(FlattenOpenIaaSTemplate)},
		{"FlattenOpenIaaSVirtualMachine", zeroOf(FlattenOpenIaaSVirtualMachine)},

		// --- Object storage -----------------------------------------------
		{"FlattenACL", zeroOf(FlattenACL)},
		{"FlattenBucket", zeroOf(FlattenBucket)},
		{"FlattenBucketFile", zeroOf(FlattenBucketFile)},
		{"FlattenObjectStorageRole", zeroOf(FlattenObjectStorageRole)},
		{"FlattenStorageAccount", zeroOf(FlattenStorageAccount)},

		// --- IAM ----------------------------------------------------------
		{"FlattenCompany", zeroOf(FlattenCompany)},
		{"FlattenFeature", zeroOf(FlattenFeature)},
		{"FlattenRole", zeroOf(FlattenRole)},
		{"FlattenTenant", zeroOf(FlattenTenant)},
		{"FlattenToken", zeroOf(FlattenToken)},
		{"FlattenUser", zeroOf(FlattenUser)},

		// --- Marketplace --------------------------------------------------
		{"FlattenMarketplaceItem", zeroOf(FlattenMarketplaceItem)},

		// --- Public Cloud VM Instances ------------------------------------
		{"FlattenPublicCloudVMRegion", zeroOf(FlattenPublicCloudVMRegion)},
		{"FlattenPublicCloudVMAvailabilityZone", zeroOf(FlattenPublicCloudVMAvailabilityZone)},
		{"FlattenPublicCloudVMFlavor", zeroOf(FlattenPublicCloudVMFlavor)},
	}
}

// knownNilSliceGaps lists the (helper, key) pairs that currently emit a typed
// nil slice on empty input because the helper passes a nil-able source field
// straight through (e.g. token.Roles, pool.Hosts) instead of normalizing it to
// []. This is SDK-tolerated today (see invariant (iii) doc), so it is a tracked
// debt, not a break. The harness pins each gap two ways:
//
//   - a helper NOT listed here must emit [] for every slice key (strict);
//   - a helper listed here must STILL emit nil for the listed key. If a helper
//     is later fixed to emit [], this test fails on the now-stale gap entry and
//     forces its removal — a ratchet that prevents the list from rotting.
//
// Keep this map shrinking. An entry is a debt to close by making the helper
// emit [] explicitly in a future test-or-fix PR, not a resting place.
var knownNilSliceGaps = map[string]map[string]bool{
	"FlattenContentLibraryItem":        {"ovf_properties": true},
	"FlattenDatastore":                 {"hosts_names": true},
	"FlattenDatastoreCluster":          {"datastores": true},
	"FlattenNetwork":                   {"host_names": true},
	"FlattenVirtualController":         {"virtual_disks": true},
	"FlattenVirtualMachine":            {"distributed_virtual_port_group_ids": true},
	"FlattenBackupOpenIaasPolicy":      {"virtual_machines": true},
	"FlattenOpenIaaSHost":              {"virtual_machines": true},
	"FlattenOpenIaaSNetwork":           {"network_adapters": true},
	"FlattenOpenIaaSPool":              {"hosts": true},
	"FlattenOpenIaaSStorageRepository": {"virtual_disks": true},
	"FlattenOpenIaaSTemplate":          {"snapshots": true, "sla_policies": true},
	"FlattenOpenIaaSVirtualMachine":    {"boot_order": true},
	"FlattenObjectStorageRole":         {"permissions": true},
	"FlattenToken":                     {"roles": true},
	"FlattenUser":                      {"source": true},
	"FlattenMarketplaceItem":           {"categories": true},
}

// TestFlattenZeroInputInvariants runs invariants (i), (ii) and (iii) over every
// registered helper. A failure on (i)/(ii) is a real datasource-read hazard: a
// bare API object (present, all fields empty) must still flatten into a
// state-safe map. A failure on (iii) is either a new un-tracked nil-slice gap
// (a helper not in knownNilSliceGaps emitting nil) or a stale gap entry (a
// helper now emitting [] for a key still listed as nil).
func TestFlattenZeroInputInvariants(t *testing.T) {
	for _, tc := range genericFlattenCases() {
		t.Run(tc.name, func(t *testing.T) {
			// (i) no panic on a zero-valued struct; (ii) non-nil map.
			out := mustNotPanic(t, tc.name, tc.build)
			if out == nil {
				t.Fatalf("%s returned a nil map on a zero-valued input; d.Set would silently drop the object", tc.name)
			}

			gaps := knownNilSliceGaps[tc.name]
			// (iii) every slice-typed value emitted from the empty source must
			// be a non-nil empty slice, never a typed nil, unless the key is a
			// declared known gap.
			for key, v := range out {
				rv := reflect.ValueOf(v)
				if !rv.IsValid() || rv.Kind() != reflect.Slice {
					continue
				}
				_, declaredGap := gaps[key]
				if rv.IsNil() && !declaredGap {
					t.Errorf("%s emits key %q as a typed nil slice on empty input; expected a non-nil empty slice ([]) to pin the present-but-empty intent. If this is intentional for now, add it to knownNilSliceGaps with a rationale.", tc.name, key)
				}
				if !rv.IsNil() && declaredGap {
					t.Errorf("%s now emits key %q as a non-nil slice, but it is still listed in knownNilSliceGaps; remove the stale gap entry.", tc.name, key)
				}
			}
			// A gap entry whose key never appears in the output is also stale.
			for key := range gaps {
				if _, present := out[key]; !present {
					t.Errorf("%s lists %q in knownNilSliceGaps but does not emit that key; remove the stale gap entry.", tc.name, key)
				}
			}
		})
	}
}

// mustNotPanic runs build and converts a panic into a test failure that names
// the offending helper, so a regression points straight at the function.
func mustNotPanic(t *testing.T, name string, build func() map[string]interface{}) (out map[string]interface{}) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked on a zero-valued struct input: %v", name, r)
		}
	}()
	return build()
}

// TestGenericFlattenRegistryIsNotEmpty guards against the registry being
// gutted to make the suite trivially pass. It also pins a lower bound so a
// future refactor that drops helpers from the registry is noticed. The bound is
// intentionally below the current count to tolerate legitimate signature
// changes without churn, while still catching a wholesale deletion.
func TestGenericFlattenRegistryIsNotEmpty(t *testing.T) {
	const minRegistered = 40
	if got := len(genericFlattenCases()); got < minRegistered {
		t.Fatalf("the generic flatten registry has shrunk to %d helpers (expected at least %d); a non-complacent suite must keep covering the helper layer", got, minRegistered)
	}
}

// The blank reference keeps the client import meaningful even if the registry
// is later trimmed during review; it also documents that the harness is tied to
// the client struct layer it protects.
var _ = client.Role{}
