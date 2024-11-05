package client

import "context"

type OpenIaaSVirtualMachineClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) VirtualMachine() *OpenIaaSVirtualMachineClient {
	return &OpenIaaSVirtualMachineClient{c.c.c}
}

type OpenIaaSVirtualMachine struct {
	ID                  string   `terraform:"id"`
	Name                string   `terraform:"name"`
	InternalID          string   `terraform:"internal_id"`
	PowerState          string   `terraform:"power_state"`
	SecureBoot          bool     `terraform:"secure_boot"`
	AutoPowerOn         bool     `terraform:"auto_power_on"`
	DvdDrive            DvdDrive `terraform:"dvd_drive"`
	BootOrder           []string `terraform:"boot_order"`
	OperatingSystemName string   `terraform:"operating_system_name"`
	CPU                 int      `terraform:"cpu"`
	NumCoresPerSocket   int      `terraform:"num_cores_per_socket"`
	Memory              int      `terraform:"memory"`
	Addresses           struct {
		IPv6 string `terraform:"ipv6"`
		IPv4 string `terraform:"ipv4"`
	} `terraform:"addresses"`
	MachineManager struct {
		ID   string `terraform:"id"`
		Name string `terraform:"name"`
	} `terraform:"machine_manager"`
	Host struct {
		ID   string `terraform:"id"`
		Name string `terraform:"name"`
	} `terraform:"host"`
	Pool struct {
		ID   string `terraform:"id"`
		Name string `terraform:"name"`
	} `terraform:"pool"`
}

type DvdDrive struct {
	Name     string `terraform:"name"`
	Attached bool   `terraform:"attached"`
}

type OpenIaaSVirtualMachineFilter struct {
	MachineManagerID string `filter:"machineManagerId"`
}

func (v *OpenIaaSVirtualMachineClient) List(
	ctx context.Context,
	filter *OpenIaaSVirtualMachineFilter) ([]*OpenIaaSVirtualMachine, error) {

	r := v.c.newRequest("GET", "/compute/v1/open_iaas/virtual_machines")
	r.addFilter(filter)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*OpenIaaSVirtualMachine
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (v *OpenIaaSVirtualMachineClient) Read(ctx context.Context, id string) (*OpenIaaSVirtualMachine, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/virtual_machines/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaaSVirtualMachine
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
