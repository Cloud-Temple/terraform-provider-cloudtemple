package provider

// Schema golden gate (issue #298).
//
// This file freezes the externally-observable *declared schema contract* of the
// provider into a committed golden JSON file and fails CI on any divergence.
// The contract is what an existing client's Terraform state may legally contain
// and whether an existing config still plans; silently changing it can break a
// client's tfstate without any change to their HCL. Line coverage proves nothing
// here, so this gate complements (it does not replace) the runtime schema-vs-
// flatten walker (#288), the resource tests, and the integration suite (#8).
//
// What the snapshot captures, per attribute (recursively, into every nested
// Elem block): the sorted key, the Type string, Required/Optional/Computed/
// ForceNew/Sensitive, MinItems/MaxItems/ConfigMode, the literal Default value
// for primitives (or has_default_func for DefaultFunc/non-primitive), the
// PRESENCE booleans has_state_func / has_diff_suppress_func /
// diff_suppress_on_refresh / has_validate_func (ValidateFunc OR ValidateDiagFunc,
// including in a scalar Elem) / has_set_func (custom TypeSet hash), the sorted
// plan-constraint lists ConflictsWith/ExactlyOneOf/AtLeastOneOf/RequiredWith,
// and the explicit Elem kind (nil / value_type:<T> / a recursed resource).
// Per resource/datasource it also captures SchemaVersion, each StateUpgrader's
// Version plus a stable cty type representation, and has_customize_diff.
//
// What it deliberately does NOT capture: Description and other human-text/doc
// fields (doc churn, not contract), and any func body, pointer or address. A
// structural snapshot only locks the PRESENCE and types of functions, never
// their behaviour: DefaultFunc/CustomizeDiff/StateFunc/Set/DiffSuppressFunc/
// validator/StateUpgrader.Upgrade behaviour changes stay covered by behavioural
// tests, and the live API shape stays covered by #8. "Near-certainty we don't
// break the tfstate" is the SUM of those layers, not this gate alone.
//
// Backward-compatibility policy. ANY golden diff fails CI; a human must classify
// it and write a justification adjacent to the golden change in the PR. v1 does
// NOT auto-classify — it fails on any diff and the human decides. Guidance only:
//   - Likely-safe (still needs a written justification, never auto-safe): a new
//     Optional / Optional+Computed attribute, a new resource/datasource. Even an
//     additive attribute can produce new state, noisy diffs, or expose
//     API-derived values, so the human must confirm.
//   - Breaking / state-shape risk (needs a StateUpgrader and/or explicit
//     approval + CHANGELOG upgrade note): removing/renaming an attribute;
//     changing Type or Elem kind; Optional->Required; adding Required; flipping
//     Computed; adding ForceNew; reducing MaxItems; flipping Sensitive; changing
//     a Default value; adding/removing/changing StateFunc/Set/DiffSuppressFunc;
//     changing a plan-constraint; bumping SchemaVersion.
//
// Human update procedure. After a deliberate, justified change:
//
//	UPDATE_SCHEMA_SNAPSHOT=1 go test ./internal/provider -run TestSchemaSnapshot
//
// Regeneration is deterministic (running it twice yields no diff). The update
// path is REFUSED in CI: if UPDATE_SCHEMA_SNAPSHOT is set while CI=true the test
// fails hard, so an automated job can never silently rebaseline the contract.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// snapshotProviderVersion is the fixed version string passed to New(...). A
// fixed value keeps the snapshot deterministic; the provider does not branch its
// schema on the version, but pinning it removes any doubt.
const snapshotProviderVersion = "schema-snapshot"

// schemaSnapshotPath is the committed golden file.
var schemaSnapshotPath = filepath.Join("testdata", "schema_snapshot.json")

// --- Serialized shapes -----------------------------------------------------
//
// Field order in these structs is irrelevant to the output: encoding/json emits
// struct fields in declaration order and sorts map keys lexically. We use
// omitempty on every optional field so a default-valued attribute stays terse
// and a future zero-valued addition does not churn every existing entry. For
// schema flags the SDK zero value (false) and "absent" are equivalent, so
// omitempty is correct and keeps the golden small.

// attributeSnapshot is the stable projection of a single *schema.Schema.
type attributeSnapshot struct {
	Type string `json:"type"`

	Required  bool `json:"required,omitempty"`
	Optional  bool `json:"optional,omitempty"`
	Computed  bool `json:"computed,omitempty"`
	ForceNew  bool `json:"force_new,omitempty"`
	Sensitive bool `json:"sensitive,omitempty"`

	MinItems   int    `json:"min_items,omitempty"`
	MaxItems   int    `json:"max_items,omitempty"`
	ConfigMode string `json:"config_mode,omitempty"`

	// Default carries the literal value for primitive defaults (string/bool/
	// int/float). HasDefaultFunc is set instead when a DefaultFunc is present or
	// the default is non-primitive. A *value* change here is breaking, so it is
	// recorded verbatim.
	Default        interface{} `json:"default,omitempty"`
	HasDefaultFunc bool        `json:"has_default_func,omitempty"`

	// Function PRESENCE only — never bodies, pointers or addresses.
	HasStateFunc          bool `json:"has_state_func,omitempty"`
	HasDiffSuppressFunc   bool `json:"has_diff_suppress_func,omitempty"`
	DiffSuppressOnRefresh bool `json:"diff_suppress_on_refresh,omitempty"`
	HasValidateFunc       bool `json:"has_validate_func,omitempty"`
	HasSetFunc            bool `json:"has_set_func,omitempty"`

	// Plan-constraint lists, sorted. They decide whether an existing config
	// still plans.
	ConflictsWith []string `json:"conflicts_with,omitempty"`
	ExactlyOneOf  []string `json:"exactly_one_of,omitempty"`
	AtLeastOneOf  []string `json:"at_least_one_of,omitempty"`
	RequiredWith  []string `json:"required_with,omitempty"`

	// Elem made explicit: "nil", "value_type:<T>" for a scalar *schema.Schema,
	// or "resource" for a nested *schema.Resource (whose body is in ElemResource).
	ElemKind     string                       `json:"elem_kind"`
	ElemResource map[string]attributeSnapshot `json:"elem_resource,omitempty"`
}

// stateUpgraderSnapshot is the stable projection of a schema.StateUpgrader. The
// type is rendered through cty's GoString, which sorts object attribute keys, so
// the representation is stable across runs (verified by the twice-no-diff check).
type stateUpgraderSnapshot struct {
	Version int    `json:"version"`
	Type    string `json:"type"`
}

// resourceSnapshot is the stable projection of a *schema.Resource (used for both
// resources and datasources).
type resourceSnapshot struct {
	SchemaVersion    int                          `json:"schema_version,omitempty"`
	HasCustomizeDiff bool                         `json:"has_customize_diff,omitempty"`
	StateUpgraders   []stateUpgraderSnapshot      `json:"state_upgraders,omitempty"`
	Schema           map[string]attributeSnapshot `json:"schema"`
}

// providerSnapshot is the whole serialized contract.
type providerSnapshot struct {
	ProviderSchema map[string]attributeSnapshot `json:"provider_schema"`
	Resources      map[string]resourceSnapshot  `json:"resources"`
	DataSources    map[string]resourceSnapshot  `json:"data_sources"`
}

// --- Serializer ------------------------------------------------------------

func serializeAttribute(s *schema.Schema) attributeSnapshot {
	a := attributeSnapshot{
		Type:                  s.Type.String(),
		Required:              s.Required,
		Optional:              s.Optional,
		Computed:              s.Computed,
		ForceNew:              s.ForceNew,
		Sensitive:             s.Sensitive,
		MinItems:              s.MinItems,
		MaxItems:              s.MaxItems,
		ConfigMode:            configModeString(s.ConfigMode),
		HasStateFunc:          s.StateFunc != nil,
		HasDiffSuppressFunc:   s.DiffSuppressFunc != nil,
		DiffSuppressOnRefresh: s.DiffSuppressOnRefresh,
		HasValidateFunc:       s.ValidateFunc != nil || s.ValidateDiagFunc != nil,
		HasSetFunc:            s.Set != nil,
		ConflictsWith:         sortedCopy(s.ConflictsWith),
		ExactlyOneOf:          sortedCopy(s.ExactlyOneOf),
		AtLeastOneOf:          sortedCopy(s.AtLeastOneOf),
		RequiredWith:          sortedCopy(s.RequiredWith),
	}

	// Default: literal value for primitives, otherwise presence-only.
	if s.DefaultFunc != nil {
		a.HasDefaultFunc = true
	} else if s.Default != nil {
		if isPrimitiveDefault(s.Default) {
			a.Default = s.Default
		} else {
			// Non-primitive literal default (rare): record presence, never the
			// value, to avoid serializing an opaque structure non-deterministically.
			a.HasDefaultFunc = true
		}
	}

	// Elem made explicit.
	switch e := s.Elem.(type) {
	case nil:
		a.ElemKind = "nil"
	case *schema.Schema:
		a.ElemKind = "value_type:" + e.Type.String()
		// A scalar Elem can still carry a validator; adding/removing it changes
		// whether an existing config is valid at plan, so fold its presence into
		// the parent attribute's has_validate_func.
		if e.ValidateFunc != nil || e.ValidateDiagFunc != nil {
			a.HasValidateFunc = true
		}
	case *schema.Resource:
		a.ElemKind = "resource"
		a.ElemResource = serializeSchemaMap(e.Schema)
	default:
		// Unknown Elem type: surface it loudly rather than silently dropping
		// contract. The mismatch will show in the golden diff.
		a.ElemKind = fmt.Sprintf("unknown:%T", e)
	}

	return a
}

// serializeSchemaMap serializes a schema map; keys are emitted into a Go map
// (encoding/json sorts them), and every value is recursively serialized.
func serializeSchemaMap(m map[string]*schema.Schema) map[string]attributeSnapshot {
	out := make(map[string]attributeSnapshot, len(m))
	for k, v := range m {
		out[k] = serializeAttribute(v)
	}
	return out
}

func serializeResource(r *schema.Resource) resourceSnapshot {
	rs := resourceSnapshot{
		SchemaVersion:    r.SchemaVersion,
		HasCustomizeDiff: r.CustomizeDiff != nil,
		Schema:           serializeSchemaMap(r.Schema),
	}
	for _, su := range r.StateUpgraders {
		rs.StateUpgraders = append(rs.StateUpgraders, stateUpgraderSnapshot{
			Version: su.Version,
			Type:    su.Type.GoString(),
		})
	}
	// StateUpgraders are ordered by ascending Version for stability regardless of
	// declaration order.
	sort.Slice(rs.StateUpgraders, func(i, j int) bool {
		return rs.StateUpgraders[i].Version < rs.StateUpgraders[j].Version
	})
	return rs
}

func serializeProvider(p *schema.Provider) providerSnapshot {
	snap := providerSnapshot{
		ProviderSchema: serializeSchemaMap(p.Schema),
		Resources:      make(map[string]resourceSnapshot, len(p.ResourcesMap)),
		DataSources:    make(map[string]resourceSnapshot, len(p.DataSourcesMap)),
	}
	for name, r := range p.ResourcesMap {
		snap.Resources[name] = serializeResource(r)
	}
	for name, d := range p.DataSourcesMap {
		snap.DataSources[name] = serializeResource(d)
	}
	return snap
}

// renderSnapshot produces sorted, pretty, deterministic JSON. encoding/json
// sorts map keys lexically and emits struct fields in declaration order, so the
// only ordering we have to impose ourselves is on the slices (constraint lists
// and StateUpgraders), which the serializer already sorts.
func renderSnapshot(p *schema.Provider) ([]byte, error) {
	snap := serializeProvider(p)
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// --- Small deterministic helpers -------------------------------------------

func sortedCopy(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}

func configModeString(m schema.SchemaConfigMode) string {
	switch m {
	case schema.SchemaConfigModeAuto:
		return "" // default; omitted from the golden
	case schema.SchemaConfigModeAttr:
		return "attr"
	case schema.SchemaConfigModeBlock:
		return "block"
	default:
		return fmt.Sprintf("unknown(%d)", int(m))
	}
}

// isPrimitiveDefault reports whether a Default value is a JSON-stable primitive
// whose literal we record. Anything else is recorded as has_default_func.
func isPrimitiveDefault(v interface{}) bool {
	switch reflect.ValueOf(v).Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// --- The gate ---------------------------------------------------------------

// TestSchemaSnapshot regenerates the schema contract from the live provider and
// compares it to the committed golden. On mismatch it fails with a readable diff
// and remediation steps. With UPDATE_SCHEMA_SNAPSHOT=1 it rewrites the golden,
// unless CI=true, in which case it fails hard so a CI job can never silently
// rebaseline the contract. It also runs InternalValidate and asserts the
// snapshot is non-empty and covers sentinel resources/datasources and a known
// nested block, so the gate cannot pass vacuously.
func TestSchemaSnapshot(t *testing.T) {
	provider := New(snapshotProviderVersion)()

	// The provider must be internally valid; an invalid schema would make the
	// snapshot meaningless.
	if err := provider.InternalValidate(); err != nil {
		t.Fatalf("Provider().InternalValidate() failed: %s", err)
	}

	current, err := renderSnapshot(provider)
	if err != nil {
		t.Fatalf("failed to render schema snapshot: %s", err)
	}

	// Meta-assertions: prove the snapshot really covers the provider, so a future
	// bug that empties it (or a vacuous serializer) is caught here, not silently.
	assertSnapshotIsNonVacuous(t, provider, current)

	update := os.Getenv("UPDATE_SCHEMA_SNAPSHOT") != ""
	inCI := os.Getenv("CI") == "true"

	if update {
		if inCI {
			t.Fatalf("UPDATE_SCHEMA_SNAPSHOT must never be used in CI (CI=true): the schema " +
				"contract cannot be blindly rebaselined by an automated job. A contract change " +
				"must be regenerated and justified by a human, locally.")
		}
		if err := os.MkdirAll(filepath.Dir(schemaSnapshotPath), 0o755); err != nil {
			t.Fatalf("failed to create testdata dir: %s", err)
		}
		if err := os.WriteFile(schemaSnapshotPath, current, 0o644); err != nil {
			t.Fatalf("failed to write golden snapshot: %s", err)
		}
		t.Logf("schema snapshot updated -> %s (%d bytes)", schemaSnapshotPath, len(current))
		return
	}

	golden, err := os.ReadFile(schemaSnapshotPath)
	if err != nil {
		t.Fatalf("failed to read golden snapshot %s: %s\n\n"+
			"If this is the first run, generate it locally with:\n"+
			"    UPDATE_SCHEMA_SNAPSHOT=1 go test ./internal/provider -run TestSchemaSnapshot",
			schemaSnapshotPath, err)
	}

	if !bytes.Equal(golden, current) {
		t.Fatalf("the declared provider schema contract changed.\n\n%s\n\n"+
			"This gate fails on ANY change to the schema that an existing client's "+
			"Terraform state depends on. Do NOT blindly regenerate. A human must:\n"+
			"  1. read the diff above and classify the change "+
			"(likely-safe additive vs. breaking/state-shape — see the policy in the "+
			"header of schema_snapshot_test.go);\n"+
			"  2. confirm a backward-compatibility decision (a breaking change needs a "+
			"StateUpgrader and/or explicit approval + a CHANGELOG upgrade note);\n"+
			"  3. write that justification in the PR, adjacent to the golden change;\n"+
			"  4. regenerate the golden locally with:\n"+
			"         UPDATE_SCHEMA_SNAPSHOT=1 go test ./internal/provider -run TestSchemaSnapshot\n"+
			"     (refused when CI=true).",
			unifiedishDiff(golden, current))
	}
}

// assertSnapshotIsNonVacuous proves the snapshot meaningfully covers the
// provider: it is non-empty, it counts the same number of resources and
// datasources the provider registers, and it contains sentinel entries plus a
// known nested block. Without these checks an empty or truncated serializer
// could "pass" against an empty golden.
func assertSnapshotIsNonVacuous(t *testing.T, provider *schema.Provider, current []byte) {
	t.Helper()

	if len(current) == 0 {
		t.Fatalf("rendered schema snapshot is empty")
	}

	var snap providerSnapshot
	if err := json.Unmarshal(current, &snap); err != nil {
		t.Fatalf("rendered snapshot is not valid JSON: %s", err)
	}

	if len(snap.ProviderSchema) == 0 {
		t.Fatalf("snapshot provider_schema is empty; expected the provider-level config block")
	}
	if got, want := len(snap.Resources), len(provider.ResourcesMap); got != want {
		t.Fatalf("snapshot covers %d resources but the provider registers %d", got, want)
	}
	if got, want := len(snap.DataSources), len(provider.DataSourcesMap); got != want {
		t.Fatalf("snapshot covers %d datasources but the provider registers %d", got, want)
	}
	if len(snap.Resources) == 0 {
		t.Fatalf("snapshot contains no resources")
	}
	if len(snap.DataSources) == 0 {
		t.Fatalf("snapshot contains no datasources")
	}

	// Sentinel resources/datasources that must always be present. These are
	// long-standing, central objects; if any of them vanished the change is
	// either a real removal (which must be human-vetted through the golden diff)
	// or a serializer regression (caught here).
	for _, name := range []string{
		"cloudtemple_compute_virtual_machine",
		"cloudtemple_iam_personal_access_token",
	} {
		if _, ok := snap.Resources[name]; !ok {
			t.Fatalf("sentinel resource %q is missing from the snapshot", name)
		}
	}
	for _, name := range []string{
		"cloudtemple_compute_virtual_machine",
		"cloudtemple_iam_features",
	} {
		if _, ok := snap.DataSources[name]; !ok {
			t.Fatalf("sentinel datasource %q is missing from the snapshot", name)
		}
	}

	// Known nested block: the iam_features datasource declares a "features" list
	// whose Elem is a nested resource (recursed). This proves the serializer
	// actually recurses into Elem blocks rather than stopping at the top level.
	features, ok := snap.DataSources["cloudtemple_iam_features"]
	if !ok {
		t.Fatalf("sentinel datasource cloudtemple_iam_features is missing")
	}
	featuresAttr, ok := features.Schema["features"]
	if !ok {
		t.Fatalf("cloudtemple_iam_features.features attribute missing from snapshot")
	}
	if featuresAttr.ElemKind != "resource" {
		t.Fatalf("cloudtemple_iam_features.features should have a nested resource Elem, got elem_kind=%q", featuresAttr.ElemKind)
	}
	if len(featuresAttr.ElemResource) == 0 {
		t.Fatalf("cloudtemple_iam_features.features nested block was not recursed into (empty elem_resource)")
	}

	// The virtual_machine resource is the one place that exercises SchemaVersion
	// and a StateUpgrader. Lock that this is reflected, so a serializer that
	// dropped StateUpgraders would be caught.
	vm, ok := snap.Resources["cloudtemple_compute_virtual_machine"]
	if !ok {
		t.Fatalf("sentinel resource cloudtemple_compute_virtual_machine is missing")
	}
	if vm.SchemaVersion == 0 || len(vm.StateUpgraders) == 0 {
		t.Fatalf("cloudtemple_compute_virtual_machine should carry SchemaVersion>0 and at least one StateUpgrader; "+
			"got schema_version=%d, %d upgraders", vm.SchemaVersion, len(vm.StateUpgraders))
	}
}

// unifiedishDiff renders a compact line-level diff between the golden and the
// freshly rendered snapshot, enough to make a mismatch readable without pulling
// in a diff dependency. It shows every differing line with a - (golden) / +
// (current) prefix.
func unifiedishDiff(golden, current []byte) string {
	goldenLines := bytes.Split(golden, []byte("\n"))
	currentLines := bytes.Split(current, []byte("\n"))

	var buf bytes.Buffer
	maxLines := len(goldenLines)
	if len(currentLines) > maxLines {
		maxLines = len(currentLines)
	}
	shown := 0
	const maxShown = 200
	for i := 0; i < maxLines; i++ {
		var g, c []byte
		if i < len(goldenLines) {
			g = goldenLines[i]
		}
		if i < len(currentLines) {
			c = currentLines[i]
		}
		if !bytes.Equal(g, c) {
			if shown >= maxShown {
				fmt.Fprintf(&buf, "... (diff truncated at %d differing lines)\n", maxShown)
				break
			}
			if i < len(goldenLines) {
				fmt.Fprintf(&buf, "- L%d: %s\n", i+1, g)
			}
			if i < len(currentLines) {
				fmt.Fprintf(&buf, "+ L%d: %s\n", i+1, c)
			}
			shown++
		}
	}
	if shown == 0 {
		return "(byte-level difference with no line-level change; check trailing whitespace/newline)"
	}
	return buf.String()
}
