package helpers

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestFlattenFeaturePreservesShapeAtRealDepth locks the state shape for the real
// API depth (2): a root with leaf children still emits an empty "subfeatures"
// list on the leaves, exactly like the historical helper. This guards against a
// silent state-shape change for existing consumers.
func TestFlattenFeaturePreservesShapeAtRealDepth(t *testing.T) {
	root := &client.Feature{ID: "r", Name: "root", SubFeatures: []*client.Feature{
		{ID: "c", Name: "child"},
	}}
	out := FlattenFeature(root)

	subs, ok := out["subfeatures"].([]map[string]interface{})
	if !ok || len(subs) != 1 {
		t.Fatalf("root must emit one subfeature, got %#v", out["subfeatures"])
	}
	child := subs[0]
	leaf, ok := child["subfeatures"].([]map[string]interface{})
	if !ok {
		t.Fatalf("a leaf at a declared level must still emit an (empty) subfeatures list, got %#v", child["subfeatures"])
	}
	if len(leaf) != 0 {
		t.Errorf("leaf subfeatures must be empty, got %#v", leaf)
	}
}

// TestFlattenFeatureIsDepthBounded proves the flatten output never carries a
// "subfeatures" key beyond the deepest declared level, whatever the tree depth.
func TestFlattenFeatureIsDepthBounded(t *testing.T) {
	l3 := &client.Feature{ID: "f3", Name: "x"}
	l2 := &client.Feature{ID: "f2", Name: "x", SubFeatures: []*client.Feature{l3}}
	l1 := &client.Feature{ID: "f1", Name: "x", SubFeatures: []*client.Feature{l2}}
	l0 := &client.Feature{ID: "f0", Name: "x", SubFeatures: []*client.Feature{l1}}

	out := FlattenFeature(l0)
	lvl1 := out["subfeatures"].([]map[string]interface{})[0]
	lvl2 := lvl1["subfeatures"].([]map[string]interface{})[0]
	if _, present := lvl2["subfeatures"]; present {
		t.Errorf("level-2 node must not carry a subfeatures key (truncation), got %#v", lvl2)
	}
	if lvl2["id"] != "f2" {
		t.Errorf("level-2 id = %v, want f2", lvl2["id"])
	}
}

// TestFlattenFeatureNilGuard ensures a nil sub-feature (a JSON null element) is
// skipped instead of panicking.
func TestFlattenFeatureNilGuard(t *testing.T) {
	root := &client.Feature{ID: "r", Name: "root", SubFeatures: []*client.Feature{
		nil,
		{ID: "c", Name: "child"},
		nil,
	}}
	out := FlattenFeature(root) // must not panic
	subs := out["subfeatures"].([]map[string]interface{})
	if len(subs) != 1 {
		t.Fatalf("nil sub-features must be skipped, expected 1 child, got %d (%#v)", len(subs), subs)
	}
	if subs[0]["id"] != "c" {
		t.Errorf("kept child id = %v, want c", subs[0]["id"])
	}
}

// TestFlattenFeatureNilRoot ensures the exported helper is nil-safe: a nil
// feature returns nil instead of panicking.
func TestFlattenFeatureNilRoot(t *testing.T) {
	if got := FlattenFeature(nil); got != nil {
		t.Errorf("FlattenFeature(nil) must return nil, got %#v", got)
	}
}

// TestFeatureExceedsDeclaredDepth locks the truncation detector that drives the
// Read warning: depth <= 3 fits, depth 4 overflows.
func TestFeatureExceedsDeclaredDepth(t *testing.T) {
	flat := &client.Feature{ID: "a", Name: "a"}
	depth2 := &client.Feature{ID: "a", SubFeatures: []*client.Feature{{ID: "b"}}}
	depth3 := &client.Feature{ID: "a", SubFeatures: []*client.Feature{
		{ID: "b", SubFeatures: []*client.Feature{{ID: "c"}}},
	}}
	depth4 := &client.Feature{ID: "a", SubFeatures: []*client.Feature{
		{ID: "b", SubFeatures: []*client.Feature{
			{ID: "c", SubFeatures: []*client.Feature{{ID: "d"}}},
		}},
	}}
	// A node at the deepest representable level whose only "children" are nil:
	// the flatten skips them, so this must NOT be flagged as a deeper tree.
	nilAtDepth := &client.Feature{ID: "a", SubFeatures: []*client.Feature{
		{ID: "b", SubFeatures: []*client.Feature{
			{ID: "c", SubFeatures: []*client.Feature{nil, nil}},
		}},
	}}

	for _, tc := range []struct {
		name string
		in   *client.Feature
		want bool
	}{
		{"nil", nil, false},
		{"flat", flat, false},
		{"depth2", depth2, false},
		{"depth3", depth3, false},
		{"depth4", depth4, true},
		{"nil children at deepest level", nilAtDepth, false},
	} {
		if got := FeatureExceedsDeclaredDepth(tc.in); got != tc.want {
			t.Errorf("%s: FeatureExceedsDeclaredDepth = %v, want %v", tc.name, got, tc.want)
		}
	}
}
