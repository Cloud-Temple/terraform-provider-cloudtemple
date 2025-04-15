package client

import "context"

type OpenIaaSMachineManagerClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) MachineManager() *OpenIaaSMachineManagerClient {
	return &OpenIaaSMachineManagerClient{c.c.c}
}

type OpenIaaSMachineManager struct {
	ID         string
	Name       string
	OSVersion  string
	OSName     string
	XOAVersion string
}

func (v *OpenIaaSMachineManagerClient) Read(ctx context.Context, id string) (*OpenIaaSMachineManager, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaaSMachineManager
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (v *OpenIaaSMachineManagerClient) List(ctx context.Context) ([]*OpenIaaSMachineManager, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas")
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out []*OpenIaaSMachineManager
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
