package main

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// probeAbsenceEndpoint is one by-id / parent-scoped read endpoint to probe with a
// deliberately-nonexistent id, to observe whether the API returns 404 (the new
// "absent -> 404" contract — safe to flip its notFoundCode from 403 to 404) or
// 403 (still 403-for-absent — do NOT flip). See issue #384.
type probeAbsenceEndpoint struct {
	name string        // stable report label, e.g. "compute.vmware.virtual_machine"
	path string        // newRequest path template, with one %s per bogus arg
	args []interface{} // bogus, well-formed-but-nonexistent argument(s)
}

// Well-formed but nonexistent identifiers. They must be syntactically VALID so the
// API performs an absence lookup (404/403) rather than route/format validation
// (which could answer 400 and tell us nothing about the absence contract): a valid
// UUID shape for id routes, valid object-storage names, a valid MAC for ReadByMAC,
// and a real marketplace provider target for the /items/%s/%s/info route.
const (
	// Random-looking valid UUIDs (NOT the nil/all-ones UUID: a stricter route
	// validator may special-case those and answer 405/400 instead of running the
	// absence lookup).
	probeBogusID     = "7f3e9a2b-1c4d-4e5f-8a6b-9c0d1e2f3a4b"
	probeBogusID2    = "b2c3d4e5-6f70-4a1b-9c2d-3e4f5a6b7c8d"
	probeBogusName   = "ctvalidateprobeabsent"
	probeBogusMAC    = "02:00:5e:10:00:01"
	probeMarketplace = "vmware"
)

// absenceProbeEndpoints is every by-id / parent-scoped read that currently uses
// requireNotFoundOrOK(resp, 403) (i.e. its GET path contains a %s). The exact
// coverage test (cycle_probe_absence_test.go) asserts this set matches a scan of
// the client package, so a new such endpoint cannot be added without a probe
// entry. Collection/aggregate endpoints (no %s) are intentionally excluded: a list
// has no "absent id" — a 403 there is a forbidden, not a missing resource — so
// #384 treats those as inherently safe to flip without a probe.
var absenceProbeEndpoints = []probeAbsenceEndpoint{
	// VMware compute
	{"compute.vmware.content_library", "/compute/v1/vcenters/content_libraries/%s", []interface{}{probeBogusID}},
	{"compute.vmware.content_library.items", "/compute/v1/vcenters/content_libraries/%s/items", []interface{}{probeBogusID}},
	{"compute.vmware.content_library.item", "/compute/v1/vcenters/content_libraries/%s/items/%s", []interface{}{probeBogusID, probeBogusID2}},
	{"compute.vmware.datastore", "/compute/v1/vcenters/datastores/%s", []interface{}{probeBogusID}},
	{"compute.vmware.datastore_cluster", "/compute/v1/vcenters/datastore_clusters/%s", []interface{}{probeBogusID}},
	{"compute.vmware.folder", "/compute/v1/vcenters/folders/%s", []interface{}{probeBogusID}},
	{"compute.vmware.host", "/compute/v1/vcenters/hosts/%s", []interface{}{probeBogusID}},
	{"compute.vmware.host_cluster", "/compute/v1/vcenters/host_clusters/%s", []interface{}{probeBogusID}},
	{"compute.vmware.network", "/compute/v1/vcenters/networks/%s", []interface{}{probeBogusID}},
	{"compute.vmware.network_adapter", "/compute/v1/vcenters/network_adapters/%s", []interface{}{probeBogusID}},
	{"compute.vmware.resource_pool", "/compute/v1/vcenters/resource_pools/%s", []interface{}{probeBogusID}},
	{"compute.vmware.virtual_controller", "/compute/v1/vcenters/virtual_controllers/%s", []interface{}{probeBogusID}},
	{"compute.vmware.virtual_datacenter", "/compute/v1/vcenters/virtual_datacenters/%s", []interface{}{probeBogusID}},
	{"compute.vmware.virtual_disk", "/compute/v1/vcenters/virtual_disks/%s", []interface{}{probeBogusID}},
	{"compute.vmware.virtual_machine", "/compute/v1/vcenters/virtual_machines/%s", []interface{}{probeBogusID}},
	{"compute.vmware.virtual_switch", "/compute/v1/vcenters/virtual_switchs/%s", []interface{}{probeBogusID}},
	{"compute.vmware.worker", "/compute/v1/vcenters/%s", []interface{}{probeBogusID}},

	// OpenIaaS compute
	{"compute.openiaas.host", "/compute/v1/open_iaas/hosts/%s", []interface{}{probeBogusID}},
	{"compute.openiaas.machine_manager", "/compute/v1/open_iaas/%s", []interface{}{probeBogusID}},
	{"compute.openiaas.network", "/compute/v1/open_iaas/networks/%s", []interface{}{probeBogusID}},
	{"compute.openiaas.network_adapter", "/compute/v1/open_iaas/network_adapters/%s", []interface{}{probeBogusID}},
	{"compute.openiaas.pool", "/compute/v1/open_iaas/pools/%s", []interface{}{probeBogusID}},
	{"compute.openiaas.replication_policy", "/compute/v1/open_iaas/replication/configurations/%s", []interface{}{probeBogusID}},
	{"compute.openiaas.replication_virtual_machine", "/compute/v1/open_iaas/replication/virtual_machines/%s/configurations", []interface{}{probeBogusID}},
	{"compute.openiaas.snapshot", "/compute/v1/open_iaas/snapshots/%s", []interface{}{probeBogusID}},
	{"compute.openiaas.storage_repository", "/compute/v1/open_iaas/storage_repositories/%s", []interface{}{probeBogusID}},
	{"compute.openiaas.template", "/compute/v1/open_iaas/templates/%s", []interface{}{probeBogusID}},
	{"compute.openiaas.virtual_disk", "/compute/v1/open_iaas/virtual_disks/%s", []interface{}{probeBogusID}},
	{"compute.openiaas.virtual_machine", "/compute/v1/open_iaas/virtual_machines/%s", []interface{}{probeBogusID}},

	// Backup
	{"backup.sla_policy", "/backup/v1/spp/policies/%s", []interface{}{probeBogusID}},
	{"backup.spp_server", "/backup/v1/spp/servers/%s", []interface{}{probeBogusID}},
	{"backup.openiaas.backup", "/backup/v1/open_iaas/backups/%s", []interface{}{probeBogusID}},
	{"backup.openiaas.policy", "/backup/v1/open_iaas/policies/%s", []interface{}{probeBogusID}},

	// Object storage
	{"object_storage.bucket", "/storage/object/v1/buckets/%s", []interface{}{probeBogusName}},
	{"object_storage.storage_account", "/storage/object/v1/storage_accounts/%s", []interface{}{probeBogusName}},

	// Marketplace
	{"marketplace.item", "/marketplace/v1/items/%s", []interface{}{probeBogusID}},
	{"marketplace.item.info", "/marketplace/v1/items/%s/%s/info", []interface{}{probeBogusID, probeMarketplace}},

	// VPC (404 already proven on dev 2026-06-26; included to RE-CONFIRM on the
	// actual recette tenant).
	{"vpc.floating_ip", "/vpc/v1/floating_ips/%s", []interface{}{probeBogusID}},
	{"vpc.private_network", "/vpc/v1/private_networks/%s", []interface{}{probeBogusID}},
	{"vpc.static_ip", "/vpc/v1/static_ips/%s", []interface{}{probeBogusID}},
	{"vpc.static_ip.by_mac", "/vpc/v1/static_ips/mac/%s", []interface{}{probeBogusMAC}},
	{"vpc.vpc", "/vpc/v1/vpc/%s", []interface{}{probeBogusID}},
}

// probeAbsenceCycle is a READ-ONLY diagnostic that GETs every by-id read endpoint
// with a deliberately-nonexistent id and records the observed HTTP status, to map
// — per endpoint, on a real tenant — whether the "absent -> 404" contract is
// deployed (#384). It is QUARANTINED (opt-in: it runs only when named explicitly
// via `-cycles probe_absence`, never under `-cycles all`), because it deliberately
// hits bogus ids and is a diagnostic, not a normal validation sweep.
//
// OPERATIONAL CAVEAT: a 403 is an unambiguous "NOT migrated" signal only when the
// probe token HAS read permission on that resource type. With an under-privileged
// token a 403 could be a genuine permission denial. Run with a broadly
// read-entitled recette token, and prefer `-runs 1 -concurrency 1` so a distressed
// endpoint does not exhaust the read-retry budget before the breaker reacts.
type probeAbsenceCycle struct{}

func (probeAbsenceCycle) Name() string { return "probe_absence" }
func (probeAbsenceCycle) Kind() Kind   { return KindRead }

// Quarantined keeps the bogus-id probe out of the `-cycles all` sweep; it runs
// only when named explicitly.
func (probeAbsenceCycle) Quarantined() bool { return true }

func (pc probeAbsenceCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	for _, ep := range absenceProbeEndpoints {
		ep := ep
		_ = r.op(pc, "probe."+ep.name, func() error {
			status, err := c.ProbeStatus(ctx, ep.path, ep.args...)
			if err != nil {
				// Transport / auth / request failure before a status: surface it
				// (categorize treats it as distress, which fails safe).
				return err
			}
			return probeAbsenceOutcome(status)
		})
	}
	return nil
}

// probeAbsenceOutcome maps an observed HTTP status to the value r.op records:
//   - 404            -> nil (OK): the desired "migrated" outcome (safe to flip).
//   - 429 / 5xx      -> a distress StatusError so the breaker trips on real API
//     distress and stops the sweep (same safety as every other cycle).
//   - everything else (403 "not migrated", or any other deterministic non-404) ->
//     a 4xx-categorised StatusError that NEVER trips the breaker, carrying the REAL
//     status in its body so the report shows it without masking the rest of the
//     map. A 2xx/3xx on a bogus id is an anomaly: it is surfaced via the body but
//     clamped to a 4xx code so it stays non-distress.
func probeAbsenceOutcome(status int) error {
	switch {
	case status == 404:
		return nil
	case status == 429 || (status >= 500 && status <= 599):
		return client.StatusError{Code: status, Body: fmt.Sprintf("absence probe: HTTP %d", status)}
	default:
		code := status
		if code < 400 || code > 499 {
			// Anomalous non-4xx (e.g. a 200/3xx on a bogus id): keep it non-distress
			// (HTTP4xx) so it does not trip the breaker; the real status is in the body.
			code = 400
		}
		return client.StatusError{Code: code, Body: fmt.Sprintf("absence probe: HTTP %d (expected 404)", status)}
	}
}
