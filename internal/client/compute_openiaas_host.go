package client

import "context"

type OpenIaaSHostClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) Host() *OpenIaaSHostClient {
	return &OpenIaaSHostClient{c.c.c}
}

type OpenIaaSHost struct {
	ID             string
	MachineManager BaseObject
	InternalId     string
	Pool           struct {
		ID   string
		Name string
		Type struct {
			Key         string
			Description string
		}
	}
	Name       string
	Master     bool
	Uptime     int
	PowerState string
	UpdateData struct {
		MaintenanceMode bool
		Status          string
	}
	RebootRequired  bool
	VirtualMachines []string
	Metrics         struct {
		XOA struct {
			Version  string
			FullName string
			Build    string
		}
		Memory struct {
			Usage int
			Size  int
		}
		Cpu struct {
			Sockets   int
			Cores     int
			Model     string
			ModelName string
		}
	}
}

type OpenIaasHostFilter struct {
	// TODO : Add filter by name
	MachineManagerId string `filter:"machineManagerId"`
	PoolId           string `filter:"poolId"`
}

func (h *OpenIaaSHostClient) List(
	ctx context.Context,
	filter *OpenIaasHostFilter) ([]*OpenIaaSHost, error) {

	r := h.c.newRequest("GET", "/compute/v1/open_iaas/hosts")
	r.addFilter(filter)
	resp, err := h.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*OpenIaaSHost
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (h *OpenIaaSHostClient) Read(ctx context.Context, id string) (*OpenIaaSHost, error) {
	r := h.c.newRequest("GET", "/compute/v1/open_iaas/hosts/%s", id)
	resp, err := h.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaaSHost
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
