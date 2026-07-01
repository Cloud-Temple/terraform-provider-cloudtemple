package provider

import (
	"reflect"
	"testing"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// datasourceFlattenCheck registers a datasource whose flatten output must
// fit its declared schema. When a flatten helper emits a key the schema
// does not declare (or with an incompatible shape), d.Set fails at runtime
// with "Invalid address to set" (#243 class) or "source data must be an
// array or slice, got map" (#241 class) and the datasource becomes unusable.
type datasourceFlattenCheck struct {
	name       string
	datasource *schema.Resource
	// rootKey is the computed list attribute for list datasources; empty
	// for single-object datasources whose Read sets the flatten map per key.
	rootKey   string
	flattened map[string]interface{}
}

// assertFlattenFitsSchema validates the flatten output against the schema
// through the real schema writer, reporting the exact offending address.
func assertFlattenFitsSchema(t *testing.T, check datasourceFlattenCheck) {
	t.Helper()
	d := schema.TestResourceDataRaw(t, check.datasource.Schema, map[string]interface{}{})

	if check.rootKey != "" {
		if _, declared := check.datasource.Schema[check.rootKey]; !declared {
			t.Errorf("%s: root key %q is not declared in the schema", check.name, check.rootKey)
			return
		}
		if err := d.Set(check.rootKey, []interface{}{check.flattened}); err != nil {
			t.Errorf("%s: the flatten output does not fit the %q schema: %s", check.name, check.rootKey, err)
		}
		return
	}

	for key, value := range check.flattened {
		if _, declared := check.datasource.Schema[key]; !declared {
			t.Errorf("%s: the flatten helper emits %q which the schema does not declare", check.name, key)
			continue
		}
		if err := d.Set(key, value); err != nil {
			t.Errorf("%s: the flatten output does not fit the schema at %q: %s", check.name, key, err)
		}
	}
}

// --- Non-zero reflection filler -------------------------------------------
//
// fillNonZero populates every settable field of a value with a non-zero,
// type-plausible value, recursing into pointers, structs (including embedded
// and anonymous ones), slices and maps. This guarantees the flatten output is
// exercised on a fully-populated client object so that a type/shape mismatch
// between a flatten helper and a datasource schema cannot hide behind a zero
// value. Coverage therefore grows automatically when a client struct gains a
// field — the opposite of hand-written fixtures, which silently leave new
// fields at zero. Validation is never triggered: d.Set on a Computed
// datasource attribute does not run ValidateFunc, so arbitrary good-typed
// values are safe for a shape test.
//
// The recursion is bounded by an absolute depth and by a self-reference cap:
// a struct type may appear at most selfRefCap times along the current path. A
// self-referential container (a struct holding a slice or pointer of its own
// type) is left empty once its element type is already nested selfRefCap deep,
// producing a finite shape instead of an unbounded tree while still exercising
// one level of nesting.
const selfRefCap = 2

// baseStruct unwraps pointers and returns the underlying struct type, or nil.
func baseStruct(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return t
	}
	return nil
}

func fillNonZero(v reflect.Value, depth int, seen map[reflect.Type]int) {
	if depth > 16 {
		return
	}
	switch v.Kind() {
	case reflect.Ptr:
		if st := baseStruct(v.Type().Elem()); st != nil && seen[st] >= selfRefCap {
			return // break a self-referential pointer chain: leave nil
		}
		if v.IsNil() {
			if !v.CanSet() {
				return
			}
			v.Set(reflect.New(v.Type().Elem()))
		}
		fillNonZero(v.Elem(), depth+1, seen)
	case reflect.Struct:
		// time.Time is a struct with unexported fields; set a fixed instant.
		if v.Type() == reflect.TypeOf(time.Time{}) {
			if v.CanSet() {
				v.Set(reflect.ValueOf(time.Unix(1700000000, 0).UTC()))
			}
			return
		}
		t := v.Type()
		seen[t]++
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !f.CanSet() { // unexported field
				continue
			}
			fillNonZero(f, depth+1, seen)
		}
		seen[t]--
	case reflect.Slice:
		if st := baseStruct(v.Type().Elem()); st != nil && seen[st] >= selfRefCap {
			v.Set(reflect.MakeSlice(v.Type(), 0, 0)) // break self-reference: empty
			return
		}
		s := reflect.MakeSlice(v.Type(), 1, 1)
		fillNonZero(s.Index(0), depth+1, seen)
		v.Set(s)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			fillNonZero(v.Index(i), depth+1, seen)
		}
	case reflect.Map:
		m := reflect.MakeMapWithSize(v.Type(), 1)
		k := reflect.New(v.Type().Key()).Elem()
		fillNonZero(k, depth+1, seen)
		val := reflect.New(v.Type().Elem()).Elem()
		fillNonZero(val, depth+1, seen)
		m.SetMapIndex(k, val)
		v.Set(m)
	case reflect.String:
		v.SetString("x")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1)
	case reflect.Interface:
		if v.IsNil() && v.CanSet() {
			v.Set(reflect.ValueOf("x"))
		}
	}
}

// filled returns a pointer to a fully-populated value of type T.
func filled[T any]() *T {
	var v T
	fillNonZero(reflect.ValueOf(&v).Elem(), 0, map[reflect.Type]int{})
	return &v
}

// flat builds the flatten output of a single-argument flatten helper applied
// to a fully-populated client object.
func flat[T any](flatten func(*T) map[string]interface{}) func() map[string]interface{} {
	return func() map[string]interface{} {
		return flatten(filled[T]())
	}
}

// flatID is like flat but mirrors the plural datasource Read paths that inject
// the object id onto each flattened element after flattening.
func flatID[T any](flatten func(*T) map[string]interface{}) func() map[string]interface{} {
	return func() map[string]interface{} {
		v := filled[T]()
		m := flatten(v)
		// The plural Reads inject the object id; mirror that. Fail loudly if this
		// helper is ever reused on a type without a string ID field, rather than
		// silently omitting the key and weakening the check.
		f := reflect.ValueOf(v).Elem().FieldByName("ID")
		if !f.IsValid() || f.Kind() != reflect.String {
			panic("flatID: type has no string ID field; use a custom builder")
		}
		m["id"] = f.String()
		return m
	}
}

// dsCoverage describes how a registered datasource is exercised by the walker:
// the flatten payload its Read produces, and the computed root list key (empty
// for single-object datasources whose Read sets the flatten map per key).
type dsCoverage struct {
	rootKey string
	build   func() map[string]interface{}
}

// datasourceCoverage maps every registered datasource to its flatten payload.
// The walker asserts the payload fits the real schema. The companion
// TestEveryDatasourceIsCoveredByTheFlattenWalker test fails if a registered
// datasource is missing here, so coverage cannot silently fall behind a newly
// added datasource. rootKey and any injected key (e.g. "id") mirror the real
// Read exactly.
var datasourceCoverage = map[string]dsCoverage{
	// --- VMware compute (vCenter) -----------------------------------------
	"cloudtemple_compute_content_libraries":       {"content_libraries", flat(helpers.FlattenContentLibrary)},
	"cloudtemple_compute_content_library":         {"", flat(helpers.FlattenContentLibrary)},
	"cloudtemple_compute_content_library_item":    {"", flat(helpers.FlattenContentLibraryItem)},
	"cloudtemple_compute_content_library_items":   {"content_library_items", flat(helpers.FlattenContentLibraryItem)},
	"cloudtemple_compute_datastore_cluster":       {"", flat(helpers.FlattenDatastoreCluster)},
	"cloudtemple_compute_datastore_clusters":      {"datastore_clusters", flat(helpers.FlattenDatastoreCluster)},
	"cloudtemple_compute_datastore":               {"", flat(helpers.FlattenDatastore)},
	"cloudtemple_compute_datastores":              {"datastores", flat(helpers.FlattenDatastore)},
	"cloudtemple_compute_folder":                  {"", flat(helpers.FlattenFolder)},
	"cloudtemple_compute_folders":                 {"folders", flat(helpers.FlattenFolder)},
	"cloudtemple_compute_guest_operating_system":  {"", flat(helpers.FlattenGuestOperatingSystem)},
	"cloudtemple_compute_guest_operating_systems": {"guest_operating_systems", flat(helpers.FlattenGuestOperatingSystem)},
	"cloudtemple_compute_host_cluster":            {"", flat(helpers.FlattenHostCluster)},
	"cloudtemple_compute_host_clusters":           {"host_clusters", flat(helpers.FlattenHostCluster)},
	"cloudtemple_compute_host":                    {"", flat(helpers.FlattenHost)},
	"cloudtemple_compute_hosts":                   {"hosts", flat(helpers.FlattenHost)},
	"cloudtemple_compute_network_adapter":         {"", flat(helpers.FlattenNetworkAdapter)},
	"cloudtemple_compute_network_adapters":        {"network_adapters", flatID(helpers.FlattenNetworkAdapter)},
	"cloudtemple_compute_network":                 {"", flat(helpers.FlattenNetwork)},
	"cloudtemple_compute_networks":                {"networks", flat(helpers.FlattenNetwork)},
	"cloudtemple_compute_resource_pool":           {"", flat(helpers.FlattenResourcePool)},
	"cloudtemple_compute_resource_pools":          {"resource_pools", flat(helpers.FlattenResourcePool)},
	"cloudtemple_compute_snapshots":               {"snapshots", flat(helpers.FlattenSnapshot)},
	"cloudtemple_compute_virtual_controllers":     {"virtual_controllers", flatID(helpers.FlattenVirtualController)},
	"cloudtemple_compute_virtual_datacenter":      {"", flat(helpers.FlattenVirtualDatacenter)},
	"cloudtemple_compute_virtual_datacenters":     {"virtual_datacenters", flat(helpers.FlattenVirtualDatacenter)},
	"cloudtemple_compute_virtual_disk":            {"", flat(helpers.FlattenVirtualDisk)},
	"cloudtemple_compute_virtual_disks":           {"virtual_disks", flatID(helpers.FlattenVirtualDisk)},
	"cloudtemple_compute_virtual_machine":         {"", flat(helpers.FlattenVirtualMachine)},
	"cloudtemple_compute_virtual_machines":        {"virtual_machines", flatID(helpers.FlattenVirtualMachine)},
	"cloudtemple_compute_virtual_switch":          {"", flat(helpers.FlattenVirtualSwitch)},
	"cloudtemple_compute_virtual_switchs":         {"virtual_switchs", flat(helpers.FlattenVirtualSwitch)},
	"cloudtemple_compute_machine_manager":         {"", flat(helpers.FlattenWorker)},
	"cloudtemple_compute_machine_managers":        {"machine_managers", flat(helpers.FlattenWorker)},

	// --- Backup (SPP) -----------------------------------------------------
	"cloudtemple_backup_job_sessions": {"job_sessions", flat(helpers.FlattenBackupJobSession)},
	"cloudtemple_backup_job":          {"", flat(helpers.FlattenBackupJob)},
	"cloudtemple_backup_jobs":         {"jobs", flat(helpers.FlattenBackupJob)},
	"cloudtemple_backup_sites":        {"sites", flat(helpers.FlattenBackupSite)},
	"cloudtemple_backup_sla_policies": {"sla_policies", flat(helpers.FlattenBackupSLAPolicy)},
	"cloudtemple_backup_sla_policy":   {"", flat(helpers.FlattenBackupSLAPolicy)},
	"cloudtemple_backup_spp_server":   {"", flat(helpers.FlattenBackupSPPServer)},
	"cloudtemple_backup_spp_servers":  {"spp_servers", flat(helpers.FlattenBackupSPPServer)},
	"cloudtemple_backup_storages":     {"storages", flat(helpers.FlattenBackupStorage)},
	"cloudtemple_backup_vcenters":     {"vcenters", flat(helpers.FlattenBackupVCenter)},
	// backup_metrics aggregates several flatten sections into distinct
	// top-level computed attributes (no single root list); mirror its Read.
	"cloudtemple_backup_metrics": {"", func() map[string]interface{} {
		return map[string]interface{}{
			"coverage":         []interface{}{helpers.FlattenBackupMetricsCoverage(filled[client.BackupMetricsCoverage]())},
			"history":          []interface{}{helpers.FlattenBackupMetricsHistory(filled[client.BackupMetricsHistory]())},
			"platform":         []interface{}{helpers.FlattenBackupMetricsPlatform(filled[client.BackupMetricsPlatform]())},
			"platform_cpu":     []interface{}{helpers.FlattenBackupMetricsPlatformCPU(filled[client.BackupMetricsPlatformCPU]())},
			"policies":         []interface{}{helpers.FlattenBackupMetricsPolicy(filled[client.BackupMetricsPolicies]())},
			"virtual_machines": []interface{}{helpers.FlattenBackupMetricsVirtualMachines(filled[client.BackupMetricsVirtualMachines]())},
		}
	}},

	// --- Backup (OpenIaaS) ------------------------------------------------
	"cloudtemple_backup_iaas_opensource_policy":   {"", flat(helpers.FlattenBackupOpenIaasPolicy)},
	"cloudtemple_backup_iaas_opensource_policies": {"policies", flat(helpers.FlattenBackupOpenIaasPolicy)},
	"cloudtemple_backup_iaas_opensource_backup":   {"", flat(helpers.FlattenBackupOpenIaasBackup)},
	"cloudtemple_backup_iaas_opensource_backups":  {"backups", flat(helpers.FlattenBackupOpenIaasBackup)},

	// --- Compute (OpenIaaS) -----------------------------------------------
	"cloudtemple_compute_iaas_opensource_host":                 {"", flat(helpers.FlattenOpenIaaSHost)},
	"cloudtemple_compute_iaas_opensource_hosts":                {"hosts", flat(helpers.FlattenOpenIaaSHost)},
	"cloudtemple_compute_iaas_opensource_storage_repository":   {"", flat(helpers.FlattenOpenIaaSStorageRepository)},
	"cloudtemple_compute_iaas_opensource_storage_repositories": {"storage_repositories", flat(helpers.FlattenOpenIaaSStorageRepository)},
	"cloudtemple_compute_iaas_opensource_pool":                 {"", flat(helpers.FlattenOpenIaaSPool)},
	"cloudtemple_compute_iaas_opensource_pools":                {"pools", flat(helpers.FlattenOpenIaaSPool)},
	"cloudtemple_compute_iaas_opensource_template":             {"", flat(helpers.FlattenOpenIaaSTemplate)},
	"cloudtemple_compute_iaas_opensource_templates":            {"templates", flatID(helpers.FlattenOpenIaaSTemplate)},
	"cloudtemple_compute_iaas_opensource_network":              {"", flat(helpers.FlattenOpenIaaSNetwork)},
	"cloudtemple_compute_iaas_opensource_networks":             {"networks", flat(helpers.FlattenOpenIaaSNetwork)},
	"cloudtemple_compute_iaas_opensource_virtual_machine":      {"", flat(helpers.FlattenOpenIaaSVirtualMachine)},
	"cloudtemple_compute_iaas_opensource_virtual_machines":     {"virtual_machines", flatID(helpers.FlattenOpenIaaSVirtualMachine)},
	"cloudtemple_compute_iaas_opensource_snapshot":             {"", flat(helpers.FlattenOpenIaaSSnapshot)},
	"cloudtemple_compute_iaas_opensource_snapshots":            {"snapshots", flat(helpers.FlattenOpenIaaSSnapshot)},
	"cloudtemple_compute_iaas_opensource_availability_zone":    {"", flat(helpers.FlattenOpenIaaSMachineManager)},
	"cloudtemple_compute_iaas_opensource_availability_zones":   {"availability_zones", flat(helpers.FlattenOpenIaaSMachineManager)},
	"cloudtemple_compute_iaas_opensource_replication_policy":   {"", flat(helpers.FlattenOpenIaaSReplicationPolicy)},
	"cloudtemple_compute_iaas_opensource_replication_policies": {"policies", flat(helpers.FlattenOpenIaaSReplicationPolicy)},
	"cloudtemple_compute_iaas_opensource_network_adapter":      {"", flat(helpers.FlattenOpenIaaSNetworkAdapter)},
	"cloudtemple_compute_iaas_opensource_network_adapters":     {"network_adapters", flatID(helpers.FlattenOpenIaaSNetworkAdapter)},
	// Both Reads pass an empty vmID (the connection-state top-level keys are a
	// resource-only path); mirror that exactly.
	"cloudtemple_compute_iaas_opensource_virtual_disk": {"", func() map[string]interface{} {
		return helpers.FlattenOpenIaaSVirtualDisk(filled[client.OpenIaaSVirtualDisk](), "")
	}},
	"cloudtemple_compute_iaas_opensource_virtual_disks": {"virtual_disks", func() map[string]interface{} {
		d := filled[client.OpenIaaSVirtualDisk]()
		m := helpers.FlattenOpenIaaSVirtualDisk(d, "")
		m["id"] = d.ID
		return m
	}},

	// --- Object storage ---------------------------------------------------
	"cloudtemple_object_storage_bucket":           {"", flat(helpers.FlattenBucket)},
	"cloudtemple_object_storage_buckets":          {"buckets", flat(helpers.FlattenBucket)},
	"cloudtemple_object_storage_bucket_files":     {"files", flat(helpers.FlattenBucketFile)},
	"cloudtemple_object_storage_storage_account":  {"", flat(helpers.FlattenStorageAccount)},
	"cloudtemple_object_storage_storage_accounts": {"storage_accounts", flat(helpers.FlattenStorageAccount)},
	"cloudtemple_object_storage_acl":              {"acls", flat(helpers.FlattenACL)},
	"cloudtemple_object_storage_role":             {"", flat(helpers.FlattenObjectStorageRole)},
	"cloudtemple_object_storage_roles":            {"roles", flat(helpers.FlattenObjectStorageRole)},

	// --- IAM --------------------------------------------------------------
	"cloudtemple_iam_company": {"", flat(helpers.FlattenCompany)},
	// Feature is self-referential and the schema declares a fixed nesting depth
	// (features -> subfeatures -> subfeatures). The generic filler stops one
	// level deep, so build a depth-4 tree by hand: a node AT the deepest declared
	// level (level 2) that itself has a child. A correct FlattenFeature must
	// truncate it so the flatten output still fits the schema; an unbounded one
	// emits an undeclared "subfeatures" key and d.Set fails (#243 class). See
	// TestIAMFeaturesFlattenIsDepthBounded for the non-complacent proof.
	"cloudtemple_iam_features": {"features", func() map[string]interface{} {
		l3 := &client.Feature{ID: "f3", Name: "x"}
		l2 := &client.Feature{ID: "f2", Name: "x", SubFeatures: []*client.Feature{l3}}
		l1 := &client.Feature{ID: "f1", Name: "x", SubFeatures: []*client.Feature{l2}}
		l0 := &client.Feature{ID: "f0", Name: "x", SubFeatures: []*client.Feature{l1}}
		return helpers.FlattenFeature(l0)
	}},
	"cloudtemple_iam_personal_access_token":  {"", flat(helpers.FlattenToken)},
	"cloudtemple_iam_personal_access_tokens": {"tokens", flat(helpers.FlattenToken)},
	"cloudtemple_iam_role":                   {"", flat(helpers.FlattenRole)},
	"cloudtemple_iam_roles":                  {"roles", flat(helpers.FlattenRole)},
	"cloudtemple_iam_tenants":                {"tenants", flat(helpers.FlattenTenant)},
	"cloudtemple_iam_user":                   {"", flat(helpers.FlattenUser)},
	"cloudtemple_iam_users":                  {"users", flat(helpers.FlattenUser)},

	// --- Marketplace ------------------------------------------------------
	"cloudtemple_marketplace_item":  {"", flat(helpers.FlattenMarketplaceItem)},
	"cloudtemple_marketplace_items": {"marketplace_items", flat(helpers.FlattenMarketplaceItem)},

	// --- VPC --------------------------------------------------------------
	// Every VPC flatten helper emits "id" itself, and the plural Reads do NOT
	// inject it afterwards (unlike the flatID datasources above); mirror that
	// with flat, not flatID.
	"cloudtemple_vpc_vpc":              {"", flat(helpers.FlattenVPC)},
	"cloudtemple_vpc_vpcs":             {"vpcs", flat(helpers.FlattenVPC)},
	"cloudtemple_vpc_private_network":  {"", flat(helpers.FlattenPrivateNetwork)},
	"cloudtemple_vpc_private_networks": {"private_networks", flat(helpers.FlattenPrivateNetwork)},
	"cloudtemple_vpc_static_ip":        {"", flat(helpers.FlattenStaticIP)},
	"cloudtemple_vpc_static_ips":       {"static_ips", flat(helpers.FlattenStaticIP)},
	"cloudtemple_vpc_floating_ip":      {"", flat(helpers.FlattenFloatingIP)},
	"cloudtemple_vpc_floating_ips":     {"floating_ips", flat(helpers.FlattenFloatingIP)},

	// --- Public Cloud VM Instances ----------------------------------------
	// FlattenPublicCloudVMRegion emits "id" itself and the plural Read does not
	// inject it afterwards; mirror that with flat, not flatID.
	"cloudtemple_public_cloud_vm_region":             {"", flat(helpers.FlattenPublicCloudVMRegion)},
	"cloudtemple_public_cloud_vm_regions":            {"regions", flat(helpers.FlattenPublicCloudVMRegion)},
	"cloudtemple_public_cloud_vm_availability_zone":  {"", flat(helpers.FlattenPublicCloudVMAvailabilityZone)},
	"cloudtemple_public_cloud_vm_availability_zones": {"availability_zones", flat(helpers.FlattenPublicCloudVMAvailabilityZone)},
	"cloudtemple_public_cloud_vm_flavor":             {"", flat(helpers.FlattenPublicCloudVMFlavor)},
	"cloudtemple_public_cloud_vm_flavors":            {"flavors", flat(helpers.FlattenPublicCloudVMFlavor)},
	"cloudtemple_public_cloud_vm_instance_family":    {"", flat(helpers.FlattenPublicCloudVMInstanceFamily)},
	"cloudtemple_public_cloud_vm_instance_families":  {"instance_families", flat(helpers.FlattenPublicCloudVMInstanceFamily)},
	"cloudtemple_public_cloud_vm_storage_type":       {"", flat(helpers.FlattenPublicCloudVMStorageType)},
	"cloudtemple_public_cloud_vm_storage_types":      {"storage_types", flat(helpers.FlattenPublicCloudVMStorageType)},
	"cloudtemple_public_cloud_vm_template":           {"", flat(helpers.FlattenPublicCloudVMTemplate)},
	"cloudtemple_public_cloud_vm_templates":          {"templates", flat(helpers.FlattenPublicCloudVMTemplate)},
	"cloudtemple_public_cloud_vm_backup_policy":      {"", flat(helpers.FlattenPublicCloudVMBackupPolicy)},
	"cloudtemple_public_cloud_vm_backup_policies":    {"backup_policies", flat(helpers.FlattenPublicCloudVMBackupPolicy)},
	"cloudtemple_public_cloud_vm_quota":              {"", flat(helpers.FlattenPublicCloudVMQuota)},
	"cloudtemple_public_cloud_vm_task":               {"", flat(helpers.FlattenPublicCloudVMTask)},
	"cloudtemple_public_cloud_vm_tasks":              {"tasks", flat(helpers.FlattenPublicCloudVMTask)},
	"cloudtemple_public_cloud_vm_instance":           {"", flat(helpers.FlattenPublicCloudVMInstance)},
	"cloudtemple_public_cloud_vm_instances":          {"instances", flat(helpers.FlattenPublicCloudVMInstance)},
	"cloudtemple_public_cloud_vm_disks":              {"disks", flat(helpers.FlattenPublicCloudVMDisk)},
	"cloudtemple_public_cloud_vm_snapshots":          {"snapshots", flat(helpers.FlattenPublicCloudVMSnapshot)},
}

// datasourceKnownGaps lists datasources deliberately NOT covered by the walker
// yet, each with its justification. A known gap is an explicit, declared
// exclusion — not a silent one. Each entry must be a registered datasource and
// must NOT also appear in datasourceCoverage. Keep this map as small as
// possible; an entry is a debt to close, not a resting place.
var datasourceKnownGaps = map[string]string{
	// Empty: every registered datasource is covered by datasourceCoverage.
	// cloudtemple_iam_features graduated from a known gap to full coverage once
	// FlattenFeature was made depth-bounded (see TestIAMFeaturesFlattenIsDepthBounded).
}

// TestDatasourceFlattenOutputsFitTheirSchemas drives the schema-vs-flatten
// walker over every registered datasource.
func TestDatasourceFlattenOutputsFitTheirSchemas(t *testing.T) {
	datasources := New("dev")().DataSourcesMap
	for name, res := range datasources {
		cov, ok := datasourceCoverage[name]
		if !ok {
			continue // reported by TestEveryDatasourceIsCoveredByTheFlattenWalker
		}
		t.Run(name, func(t *testing.T) {
			assertFlattenFitsSchema(t, datasourceFlattenCheck{
				name:       name,
				datasource: res,
				rootKey:    cov.rootKey,
				flattened:  cov.build(),
			})
		})
	}
}

// TestEveryDatasourceIsCoveredByTheFlattenWalker fails if a registered
// datasource is missing from datasourceCoverage (coverage must grow with every
// new datasource), or if the registry references a datasource that is no longer
// registered.
func TestEveryDatasourceIsCoveredByTheFlattenWalker(t *testing.T) {
	datasources := New("dev")().DataSourcesMap
	for name := range datasources {
		_, covered := datasourceCoverage[name]
		_, known := datasourceKnownGaps[name]
		if !covered && !known {
			t.Errorf("datasource %q is neither covered by the flatten/schema walker nor a declared known gap; add it to datasourceCoverage (or, with justification, to datasourceKnownGaps)", name)
		}
		if covered && known {
			t.Errorf("datasource %q is both covered and a declared known gap; remove it from one", name)
		}
	}
	for name := range datasourceCoverage {
		if _, ok := datasources[name]; !ok {
			t.Errorf("datasourceCoverage references %q which is not a registered datasource", name)
		}
	}
	for name := range datasourceKnownGaps {
		if _, ok := datasources[name]; !ok {
			t.Errorf("datasourceKnownGaps references %q which is not a registered datasource", name)
		}
	}
}

// TestIAMFeaturesFlattenIsDepthBounded proves FlattenFeature never breaks the
// cloudtemple_iam_features read, even when the feature tree is deeper than the
// schema declares. The schema declares "subfeatures" at two nesting levels; its
// deepest element declares only {id,name}. An unbounded flatten (the historical
// helper) emits a "subfeatures" key on a node at that deepest level as soon as
// the node has children, and d.Set then fails with "Invalid address to set"
// (#243 class), making the WHOLE datasource unusable.
//
// Depth 4 is the first discriminating case: at depth 3 the deepest node is a
// leaf and the historical helper only writes an empty list under the undeclared
// key, which the SDK writer tolerates; depth 4 forces a NON-empty list there.
// This test fails (panics/errors in d.Set) with an unbounded FlattenFeature and
// passes with the depth-bounded one.
func TestIAMFeaturesFlattenIsDepthBounded(t *testing.T) {
	// root -> child -> grandchild -> great-grandchild (4 node levels).
	l3 := &client.Feature{ID: "f3", Name: "great-grandchild"}
	l2 := &client.Feature{ID: "f2", Name: "grandchild", SubFeatures: []*client.Feature{l3}}
	l1 := &client.Feature{ID: "f1", Name: "child", SubFeatures: []*client.Feature{l2}}
	l0 := &client.Feature{ID: "f0", Name: "root", SubFeatures: []*client.Feature{l1}}

	flattened := helpers.FlattenFeature(l0)

	// (1) The flatten output must fit the real schema (no read-breaking crash).
	d := schema.TestResourceDataRaw(t, dataSourceFeatures().Schema, map[string]interface{}{})
	if err := d.Set("features", []interface{}{flattened}); err != nil {
		t.Fatalf("depth-4 flatten output does not fit the iam_features schema: %s", err)
	}

	// (2) Every representable level (0, 1, 2) is preserved.
	for path, want := range map[string]string{
		"features.0.id":                             "f0",
		"features.0.subfeatures.0.id":               "f1",
		"features.0.subfeatures.0.subfeatures.0.id": "f2",
	} {
		if got := d.Get(path); got != want {
			t.Errorf("%s = %v, want %q", path, got, want)
		}
	}

	// (3) The deepest declared level must NOT carry a "subfeatures" key: the
	// great-grandchild (level 3) is truncated. Emitting it is exactly the bug.
	level1 := flattened["subfeatures"].([]map[string]interface{})[0]
	level2 := level1["subfeatures"].([]map[string]interface{})[0]
	if _, present := level2["subfeatures"]; present {
		t.Errorf("level-2 node must not carry a 'subfeatures' key (it would break d.Set); got %#v", level2)
	}
}

// schemaSubfeatureNesting returns how many nested "subfeatures" levels the
// cloudtemple_iam_features schema declares under "features".
func schemaSubfeatureNesting(res *schema.Resource) int {
	depth := 0
	cur := res.Schema["features"]
	for cur != nil {
		elem, ok := cur.Elem.(*schema.Resource)
		if !ok {
			break
		}
		sub, ok := elem.Schema["subfeatures"]
		if !ok {
			break
		}
		depth++
		cur = sub
	}
	return depth
}

// TestIAMFeaturesSchemaAndFlattenDepthAgree ties the FlattenFeature emission
// depth to the SCHEMA itself (introspected), not to a hardcoded constant. If a
// future change extends the schema's "subfeatures" nesting without bumping the
// flatten bound (or the reverse), the flatten would silently truncate too early
// or overflow and crash; this test catches that drift instead of letting the
// hand-maintained coupling rot.
func TestIAMFeaturesSchemaAndFlattenDepthAgree(t *testing.T) {
	schemaDepth := schemaSubfeatureNesting(dataSourceFeatures())
	if schemaDepth < 1 {
		t.Fatalf("expected the iam_features schema to declare nested subfeatures, got depth %d", schemaDepth)
	}

	// A chain deeper than the schema, so the flatten bound is what limits it.
	var build func(n int) *client.Feature
	build = func(n int) *client.Feature {
		f := &client.Feature{ID: "x", Name: "x"}
		if n > 0 {
			f.SubFeatures = []*client.Feature{build(n - 1)}
		}
		return f
	}
	flattened := helpers.FlattenFeature(build(schemaDepth + 2))

	// Count the "subfeatures" levels the flatten actually emits.
	emitted := 0
	for node := flattened; ; {
		subs, ok := node["subfeatures"].([]map[string]interface{})
		if !ok || len(subs) == 0 {
			break
		}
		emitted++
		node = subs[0]
	}
	if emitted != schemaDepth {
		t.Errorf("flatten emits subfeatures at %d levels but the schema declares %d; the flatten bound (maxFeatureSubNesting) and the schema disagree", emitted, schemaDepth)
	}

	// The deep flatten must also fit the schema without crashing.
	d := schema.TestResourceDataRaw(t, dataSourceFeatures().Schema, map[string]interface{}{})
	if err := d.Set("features", []interface{}{flattened}); err != nil {
		t.Errorf("deep flatten output does not fit the iam_features schema: %s", err)
	}
}
