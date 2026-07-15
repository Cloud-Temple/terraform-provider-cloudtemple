package provider

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// newEmptyResourceData returns a ResourceData backed by an empty schema. The
// functions under test only touch SetId/Id, which are schema-independent.
func newEmptyResourceData() *schema.ResourceData {
	return (&schema.Resource{Schema: map[string]*schema.Schema{}}).TestResourceData()
}

// TestSetIdFromActivityState pins how a resource id is resolved from an async
// activity's State (#293, S7). Setting the wrong id makes Terraform track the
// wrong remote object — a state-safety hazard. The function must set the id
// ONLY when there is exactly one state carrying a non-empty Result, and must
// never guess (ambiguous) or clobber on a no-op. Non-complacent: dropping the
// "exactly one" guard reds the multi-state case.
func TestSetIdFromActivityState(t *testing.T) {
	tests := []struct {
		name     string
		activity *client.Activity
		priorID  string
		wantID   string
	}{
		{
			name:     "nil activity is a no-op",
			activity: nil,
			wantID:   "",
		},
		{
			name:     "empty state is a no-op",
			activity: &client.Activity{State: map[string]client.ActivityState{}},
			wantID:   "",
		},
		{
			name: "exactly one state with a result sets the id",
			activity: &client.Activity{State: map[string]client.ActivityState{
				"step-1": {Result: "vm-123"},
			}},
			wantID: "vm-123",
		},
		{
			name: "exactly one state with an empty result is a no-op",
			activity: &client.Activity{State: map[string]client.ActivityState{
				"step-1": {Result: ""},
			}},
			wantID: "",
		},
		{
			name: "more than one state is a no-op (ambiguous, never guess the id)",
			activity: &client.Activity{State: map[string]client.ActivityState{
				"step-1": {Result: "vm-1"},
				"step-2": {Result: "vm-2"},
			}},
			wantID: "",
		},
		{
			name:     "a nil activity never clobbers a pre-existing id",
			activity: nil,
			priorID:  "pre-existing",
			wantID:   "pre-existing",
		},
		{
			name: "exactly one state with an empty result never clobbers a pre-existing id",
			activity: &client.Activity{State: map[string]client.ActivityState{
				"step-1": {Result: ""},
			}},
			priorID: "pre-existing",
			wantID:  "pre-existing",
		},
		{
			name: "more than one state never clobbers a pre-existing id",
			activity: &client.Activity{State: map[string]client.ActivityState{
				"step-1": {Result: "vm-1"},
				"step-2": {Result: "vm-2"},
			}},
			priorID: "pre-existing",
			wantID:  "pre-existing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newEmptyResourceData()
			if tt.priorID != "" {
				d.SetId(tt.priorID)
			}
			setIdFromActivityState(d, tt.activity)
			if d.Id() != tt.wantID {
				t.Fatalf("id = %q, want %q", d.Id(), tt.wantID)
			}
		})
	}
}

// TestSetIdFromActivityConcernedItems pins how a resource id is resolved from
// an activity's ConcernedItems (#293, S7). The id must come from the item
// matching the expected type — regardless of its position — and the function
// must never adopt a wrong-typed or empty id, never panic when no item matches,
// and never clobber on a no-op. Non-complacent: dropping the index guard panics
// the no-match case; ignoring the type filter reds the position case; dropping
// the empty-id guard reds the empty-id case.
func TestSetIdFromActivityConcernedItems(t *testing.T) {
	tests := []struct {
		name         string
		activity     *client.Activity
		expectedType string
		priorID      string
		wantID       string
	}{
		{
			name:         "nil activity is a no-op",
			activity:     nil,
			expectedType: "virtual_machine",
			wantID:       "",
		},
		{
			name:         "empty concerned items is a no-op",
			activity:     &client.Activity{ConcernedItems: nil},
			expectedType: "virtual_machine",
			wantID:       "",
		},
		{
			name: "the item matching the expected type sets the id",
			activity: &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
				{ID: "vm-1", Type: "virtual_machine"},
			}},
			expectedType: "virtual_machine",
			wantID:       "vm-1",
		},
		{
			name: "no item of the expected type is a no-op (never adopt a wrong-typed id)",
			activity: &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
				{ID: "net-1", Type: "network"},
			}},
			expectedType: "virtual_machine",
			wantID:       "",
		},
		{
			name: "the expected type is selected regardless of position (not index 0)",
			activity: &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
				{ID: "net-1", Type: "network"},
				{ID: "vm-1", Type: "virtual_machine"},
			}},
			expectedType: "virtual_machine",
			wantID:       "vm-1",
		},
		{
			name: "the first item of the expected type wins on duplicates",
			activity: &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
				{ID: "vm-1", Type: "virtual_machine"},
				{ID: "vm-2", Type: "virtual_machine"},
			}},
			expectedType: "virtual_machine",
			wantID:       "vm-1",
		},
		{
			name: "a no match never clobbers a pre-existing id",
			activity: &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
				{ID: "net-1", Type: "network"},
			}},
			expectedType: "virtual_machine",
			priorID:      "pre-existing",
			wantID:       "pre-existing",
		},
		{
			name: "a matching item with an empty id is a no-op (never clobber with an empty id)",
			activity: &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
				{ID: "", Type: "virtual_machine"},
			}},
			expectedType: "virtual_machine",
			priorID:      "pre-existing",
			wantID:       "pre-existing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newEmptyResourceData()
			if tt.priorID != "" {
				d.SetId(tt.priorID)
			}
			setIdFromActivityConcernedItems(d, tt.activity, tt.expectedType)
			if d.Id() != tt.wantID {
				t.Fatalf("id = %q, want %q", d.Id(), tt.wantID)
			}
		})
	}
}
