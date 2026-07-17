package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// isUUID reports whether s is a valid UUID, reusing the exact validator the
// schemas use (validation.IsUUID) so the parse and the schema agree.
func isUUID(s string) bool {
	_, errs := validation.IsUUID(s, "id")
	return len(errs) == 0
}

// sameUUID compares two UUIDs case-insensitively: validation.IsUUID accepts
// upper- and lower-case hex, and an id written upper-case (import/config) must
// match the platform's canonical lower-case id — otherwise an equality guard
// (e.g. "candidate != vmID" or "still listed") could silently misfire.
func sameUUID(a, b string) bool {
	return strings.EqualFold(a, b)
}

// isStatusCode reports whether err is a client.StatusError with the given code.
func isStatusCode(err error, code int) bool {
	var se client.StatusError
	return errors.As(err, &se) && se.Code == code
}

// activityConcernedItemID returns the id of the (first) concerned item of the
// given type, or "" if none. NON-mutating (unlike setIdFromActivityConcernedItems),
// so the caller can validate the candidate before adopting it as the resource id.
func activityConcernedItemID(a *client.Activity, expectedType string) string {
	if a == nil {
		return ""
	}
	for _, ci := range a.ConcernedItems {
		if ci.Type == expectedType {
			return ci.ID
		}
	}
	return ""
}

// parseVMScopedID splits a composite "<virtual_machine_id>/<childID>" import id
// into its two parts. It requires EXACTLY two non-empty UUID parts; any other
// shape is an error, so a malformed import id never produces a partial or wrong
// state (E0-5).
func parseVMScopedID(id string) (vmID, childID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf(`invalid import id %q: expected the composite form "<virtual_machine_id>/<id>"`, id)
	}
	if !isUUID(parts[0]) {
		return "", "", fmt.Errorf(`invalid import id %q: %q is not a valid virtual machine UUID`, id, parts[0])
	}
	if !isUUID(parts[1]) {
		return "", "", fmt.Errorf(`invalid import id %q: %q is not a valid UUID`, id, parts[1])
	}
	return parts[0], parts[1], nil
}

// formatVMScopedID builds the composite import id for a VM-scoped child resource.
func formatVMScopedID(vmID, childID string) string {
	return vmID + "/" + childID
}

// importVMScopedResource is the StateContext importer for a VM-scoped child
// resource: it parses "<vmID>/<childID>", sets virtual_machine_id + the resource
// id (the child id), and lets the subsequent Read populate the rest. The child's
// Read must re-derive vmID from virtual_machine_id on every refresh (the GET path
// nests it), not only at import.
func importVMScopedResource() schema.StateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
		vmID, childID, err := parseVMScopedID(d.Id())
		if err != nil {
			return nil, err
		}
		if err := d.Set("virtual_machine_id", vmID); err != nil {
			return nil, err
		}
		d.SetId(childID)
		return []*schema.ResourceData{d}, nil
	}
}
