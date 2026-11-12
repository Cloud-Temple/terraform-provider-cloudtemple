package client

import (
	"context"
	"fmt"
	"math"
)

// VAppSnapshot is intentionally separate from the general VM read model. An
// absent/partial property collection must never become evidence for deletion.
// Raw descriptors retain metadata unknown to Terraform, including unrelated
// non-string properties. The update path validates only descriptors it replays.
type VAppSnapshot struct {
	ID         string                   `json:"id"`
	PowerState string                   `json:"powerState"`
	Enabled    bool                     `json:"vAppEnabled"`
	Properties []map[string]interface{} `json:"vAppProperties"`
}

func (v *VirtualMachineClient) ReadVAppSnapshot(ctx context.Context, id string) (*VAppSnapshot, error) {
	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_machines/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireHttpCodes(resp, 200); err != nil {
		return nil, err
	}
	var out VAppSnapshot
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}
	if out.ID != id || out.Properties == nil {
		return nil, fmt.Errorf("incomplete vApp snapshot")
	}
	keys := map[float64]bool{}
	for _, p := range out.Properties {
		// Every textual identity must be known before a caller can prove that
		// a requested property is absent. Do not normalize unrelated legacy IDs.
		propertyID, ok := p["id"].(string)
		if !ok || propertyID == "" {
			return nil, fmt.Errorf("missing or invalid vApp textual identity")
		}
		key, ok := p["key"].(float64)
		if !ok || key < 0 || key > math.MaxInt32 || math.Trunc(key) != key || keys[key] {
			return nil, fmt.Errorf("invalid or duplicate vApp numeric key")
		}
		keys[key] = true
	}
	return &out, nil
}

func (v *VirtualMachineClient) AddVAppProperty(ctx context.Context, id string, property map[string]interface{}) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/vcenters/virtual_machines/%s/vapp_properties", id)
	r.obj = []map[string]interface{}{property}
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *VirtualMachineClient) DeleteVAppProperty(ctx context.Context, id string, key int) (string, error) {
	r := v.c.newRequest("DELETE", "/compute/v1/vcenters/virtual_machines/%s/vapp_properties", id)
	r.obj = struct {
		Keys []int `json:"keys"`
	}{[]int{key}}
	return v.c.doRequestAndReturnActivity(ctx, r)
}
