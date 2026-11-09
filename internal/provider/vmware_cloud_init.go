package provider

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const cloudInitRecovery = "The VM may have partial or duplicate vApp properties. Keep it stopped. Do not reapply or power it on until all backend activities/tasks have finished and the properties have been reconciled. Terraform retains the previous cloud_init intent; this is not a backend rollback."

type cloudInitEdit struct {
	original    map[string]interface{}
	replacement map[string]interface{}
}

// A separate, stopped-only update avoids changing creation semantics or running
// hardware/power handlers while duplicate property IDs temporarily exist.
func updateVMwareCloudInit(ctx context.Context, d *schema.ResourceData, c *client.Client) (diags diag.Diagnostics) {
	old, newValue := d.GetChange("cloud_init")
	id := d.Id()
	defer func() {
		if diags.HasError() {
			d.SetId(id)
			if err := d.Set("cloud_init", old); err != nil {
				diags = append(diags, diag.Errorf("could not restore previous cloud_init intent")...)
			}
		}
	}()
	for name, s := range resourceVirtualMachine().Schema {
		if name != "cloud_init" && (s.Optional || s.Required) && d.HasChange(name) {
			return refuseBeforeAnyMutation(d, diag.Errorf("apply cloud_init changes separately from other VM changes; apply power_state = off first"))
		}
	}
	before, err := c.Compute().VirtualMachine().ReadVAppSnapshot(ctx, id)
	if err != nil {
		return refuseBeforeAnyMutation(d, diag.Errorf("cannot obtain a complete vApp snapshot; cloud_init was not changed"))
	}
	edits, err := planCloudInitEdits(before, old.(map[string]interface{}), newValue.(map[string]interface{}), d.Get("deploy_options").(map[string]interface{}))
	if err != nil {
		return refuseBeforeAnyMutation(d, diag.Errorf("cannot update cloud_init: %s", err))
	}
	expected := before
	for _, edit := range edits {
		if err := verifyVAppSnapshot(ctx, c, id, expected); err != nil {
			return cloudInitUpdateError("pre-write verification", "")
		}
		if edit.replacement != nil {
			activity, err := c.Compute().VirtualMachine().AddVAppProperty(ctx, id, edit.replacement)
			if err != nil {
				return cloudInitUpdateError("adding property", activity)
			}
			// A silent waiter prevents backend-echoed cloud-init from leaking into debug logs.
			if _, err = c.Activity().WaitForCompletion(ctx, activity, nil); err != nil {
				return cloudInitUpdateError("waiting for property addition", activity)
			}
			actual, err := c.Compute().VirtualMachine().ReadVAppSnapshot(ctx, id)
			if err != nil {
				return cloudInitUpdateError("reading added property", activity)
			}
			if err = verifyVAppAddition(expected, actual, edit.replacement); err != nil {
				return cloudInitUpdateError("verifying added property", activity)
			}
			expected = actual
		}
		if edit.original != nil {
			if err := verifyVAppSnapshot(ctx, c, id, expected); err != nil {
				return cloudInitUpdateError("pre-delete verification", "")
			}
			key := int(edit.original["key"].(float64))
			activity, err := c.Compute().VirtualMachine().DeleteVAppProperty(ctx, id, key)
			if err != nil {
				return cloudInitUpdateError("deleting original property", activity)
			}
			// A silent waiter prevents backend-echoed cloud-init from leaking into debug logs.
			if _, err = c.Activity().WaitForCompletion(ctx, activity, nil); err != nil {
				return cloudInitUpdateError("waiting for original property deletion", activity)
			}
			remaining := make([]map[string]interface{}, 0, len(expected.Properties)-1)
			for _, p := range expected.Properties {
				if p["key"] != float64(key) {
					remaining = append(remaining, p)
				}
			}
			expected = &client.VAppSnapshot{ID: id, PowerState: "stopped", Enabled: true, Properties: remaining}
			if err := verifyVAppSnapshot(ctx, c, id, expected); err != nil {
				return cloudInitUpdateError("verifying original property deletion", activity)
			}
		}
	}
	// General read still maps the established schema; it does not hydrate or
	// rewrite cloud_init for existing resources. Confirm vApp data again afterwards.
	if diags = computeVirtualMachineRead(ctx, d, c); diags.HasError() {
		return cloudInitUpdateError("final VM read", "")
	}
	if err := verifyVAppSnapshot(ctx, c, id, expected); err != nil {
		return cloudInitUpdateError("final vApp verification", "")
	}
	return diags
}

func cloudInitUpdateError(stage, activity string) diag.Diagnostics {
	// Never echo API errors, property values or descriptors: cloud-init can contain
	// credentials, and API error messages may include the rejected request body.
	return diag.Errorf("cloud_init update failed during %s (activity %q). %s", stage, activity, cloudInitRecovery)
}

func planCloudInitEdits(vm *client.VAppSnapshot, old, desired, deploy map[string]interface{}) ([]cloudInitEdit, error) {
	if vm.PowerState != "stopped" || !vm.Enabled {
		return nil, fmt.Errorf("VM must already be stopped with vApp enabled; apply power_state = off separately first")
	}
	keys := map[string]bool{}
	for k, v := range old {
		if want, ok := desired[k]; !ok || want != v {
			keys[k] = true
		}
	}
	for k, v := range desired {
		if was, ok := old[k]; !ok || was != v {
			keys[k] = true
		}
	}
	ordered := make([]string, 0, len(keys))
	for k := range keys {
		ordered = append(ordered, k)
	}
	sort.Strings(ordered)
	edits := []cloudInitEdit{}
	for _, id := range ordered {
		if !transparentVAppString(id) || utf8.RuneCountInString(id) > 60 {
			return nil, fmt.Errorf("a changed property ID cannot be transmitted without API normalization")
		}
		if _, ok := deploy[id]; ok {
			return nil, fmt.Errorf("a changed cloud_init key also belongs to deploy_options; resolve ownership first")
		}
		var live map[string]interface{}
		for _, p := range vm.Properties {
			if p["id"] == id {
				if live != nil {
					return nil, fmt.Errorf("duplicate changed vApp property IDs require manual reconciliation")
				}
				live = p
			}
		}
		was, managed := old[id]
		want, present := desired[id]
		if managed && live == nil {
			return nil, fmt.Errorf("a previously managed property is missing; reconcile it before applying (metadata cannot be reconstructed)")
		}
		if !managed && live != nil {
			return nil, fmt.Errorf("a newly managed key already exists in vApp; refusing to adopt or overwrite it")
		}
		if live != nil && live["value"] != was && (!present || live["value"] != want) {
			return nil, fmt.Errorf("a changed vApp property has drifted; reconcile it before applying")
		}
		if present && live != nil && live["value"] == want {
			continue
		}
		edit := cloudInitEdit{original: live}
		if present {
			value, ok := want.(string)
			if !ok || !transparentVAppString(value) {
				return nil, fmt.Errorf("a changed cloud_init value is empty or would be trimmed/HTML-escaped by the API")
			}
			var payload map[string]interface{}
			if live == nil {
				payload = map[string]interface{}{"id": id, "label": id, "type": "string", "userConfigurable": true}
			} else {
				var err error
				// The old value is not replayed. Validate the desired value and
				// preserved metadata, including legacy values only when retained.
				clone := make(map[string]interface{}, len(live))
				for k, v := range live {
					clone[k] = v
				}
				clone["value"] = value
				payload, err = vAppPayload(clone)
				if err != nil {
					return nil, err
				}
			}
			payload["value"] = value
			edit.replacement = payload
		}
		edits = append(edits, edit)
	}
	return edits, nil
}

// The public API trims and HTML-escapes all strings, including '/'. Reject a
// non-round-trippable request before any write instead of persisting false intent.
func transparentVAppString(s string) bool {
	return s != "" && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "&<>\"'/`\\")
}

// Normalize only documented empty metadata and 0/1 booleans. Unknown metadata
// prevents replay; unrelated properties bypass this function and remain exact.
func vAppPayload(p map[string]interface{}) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	for k, v := range p {
		switch k {
		case "key":
			continue
		case "typeReference":
			if v == nil || v == float64(0) || v == "" {
				continue
			}
			return nil, fmt.Errorf("vApp typeReference cannot be preserved by the public API")
		case "userConfigurable":
			switch v {
			case true, float64(1):
				out[k] = true
			case false, float64(0):
				out[k] = false
			default:
				return nil, fmt.Errorf("invalid vApp userConfigurable metadata")
			}
		case "id", "label", "type", "classId", "instanceId", "category", "description", "defaultValue", "value":
			required := k == "id" || k == "label" || k == "type"
			if !required && (v == nil || v == float64(0) || v == "") {
				continue
			}
			s, ok := v.(string)
			if !ok || !transparentVAppString(s) {
				return nil, fmt.Errorf("vApp metadata cannot be replayed without API normalization")
			}
			if k != "value" && k != "defaultValue" && utf8.RuneCountInString(s) > 60 {
				return nil, fmt.Errorf("vApp metadata exceeds the public API limit")
			}
			out[k] = s
		default:
			return nil, fmt.Errorf("unknown vApp metadata cannot be preserved by the public API")
		}
	}
	if out["id"] == nil || out["label"] == nil || (out["type"] != "string" && out["type"] != "password") {
		return nil, fmt.Errorf("cloud_init updates require a labelled string or password vApp property")
	}
	return out, nil
}

func vAppPropertiesByKey(vm *client.VAppSnapshot) map[interface{}]map[string]interface{} {
	out := map[interface{}]map[string]interface{}{}
	for _, p := range vm.Properties {
		out[p["key"]] = p
	}
	return out
}

func sameVAppSnapshot(a, b *client.VAppSnapshot) bool {
	return a.ID == b.ID && b.PowerState == "stopped" && b.Enabled && reflect.DeepEqual(vAppPropertiesByKey(a), vAppPropertiesByKey(b))
}

func verifyVAppSnapshot(ctx context.Context, c *client.Client, id string, expected *client.VAppSnapshot) error {
	actual, err := c.Compute().VirtualMachine().ReadVAppSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if !sameVAppSnapshot(expected, actual) {
		return fmt.Errorf("vApp snapshot changed")
	}
	return nil
}

func verifyVAppAddition(before, after *client.VAppSnapshot, payload map[string]interface{}) error {
	if before.ID != after.ID || after.PowerState != "stopped" || !after.Enabled || len(after.Properties) != len(before.Properties)+1 {
		return fmt.Errorf("unexpected addition result")
	}
	old := vAppPropertiesByKey(before)
	found := false
	for _, p := range after.Properties {
		if original, ok := old[p["key"]]; ok {
			if !reflect.DeepEqual(original, p) {
				return fmt.Errorf("existing descriptor changed")
			}
			continue
		}
		actual, err := vAppPayload(p)
		if err != nil || found || !reflect.DeepEqual(actual, payload) {
			return fmt.Errorf("added descriptor differs")
		}
		found = true
	}
	if !found {
		return fmt.Errorf("added descriptor missing")
	}
	return nil
}
