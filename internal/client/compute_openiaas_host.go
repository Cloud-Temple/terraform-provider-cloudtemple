package client

import "context"

type OpenIaaSHostClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) Host() *OpenIaaSHostClient {
	return &OpenIaaSHostClient{c.c.c}
}

type OpenIaaSHost struct {
	ID             string `terraform:"id"`
	MachineManager struct {
		ID   string `terraform:"id"`
		Name string `terraform:"name"`
		Type string `terraform:"type"`
	} `terraform:"machine_manager_id"`
	InternalId string `terraform:"internal_id"`
	Pool       struct {
		ID   string `terraform:"id"`
		Name string `terraform:"name"`
		Type struct {
			Key         string `terraform:"key"`
			Description string `terraform:"description"`
		} `terraform:"type"`
	} `terraform:"pool"`
	Name       string `terraform:"name"`
	Master     bool   `terraform:"master"`
	Uptime     int    `terraform:"uptime"`
	PowerState string `terraform:"power_state"`
	UpdateData struct {
		MaintenanceMode bool   `terraform:"maintenance_mode"`
		Status          string `terraform:"status"`
	} `terraform:"update_data"`
	RebootRequired  bool     `terraform:"reboot_required"`
	VirtualMachines []string `terraform:"virtual_machines"`
	Metrics         struct {
		XOA struct {
			Version  string `terraform:"version"`
			FullName string `terraform:"full_name"`
			Build    string `terraform:"build"`
		} `terraform:"xoa"`
		Memory struct {
			Usage int `terraform:"usage"`
			Size  int `terraform:"size"`
		} `terraform:"memory"`
		Cpu struct {
			Sockets   int    `terraform:"sockets"`
			Cores     int    `terraform:"cores"`
			Model     string `terraform:"model"`
			ModelName string `terraform:"model_name"`
		} `terraform:"cpu"`
	} `terraform:"metrics"`
}

type OpenIaasHostFilter struct {
	// TODO : Add filter by name
	MachineManagerId string `filter:"machineManagerId"`
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
