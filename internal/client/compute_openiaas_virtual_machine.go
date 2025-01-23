package client

import (
	"context"
	"fmt"
	"time"

	"github.com/sethvargo/go-retry"
)

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
	Tools               struct {
		Detected bool   `terraform:"detected"`
		Version  string `terraform:"version"`
	} `terraform:"tools"`
	Addresses struct {
		IPv6 string `terraform:"ipv6"`
		IPv4 string `terraform:"ipv4"`
	} `terraform:"addresses"`
	MachineManager struct {
		ID   string `terraform:"id"`
		Name string `terraform:"name"`
		Type string `terraform:"type"`
	} `terraform:"machine_manager"`
	Host BaseObject `terraform:"host"`
	Pool BaseObject `terraform:"pool"`
}

type DvdDrive struct {
	Name     string `terraform:"name"`
	Attached bool   `terraform:"attached"`
}

type OpenIaaSVirtualMachineFilter struct {
	MachineManagerID string `filter:"machineManagerId"`
}

type CreateOpenIaasVirtualMachineRequest struct {
	Name       string `json:"name"`
	TemplateID string `json:"templateId"`
	CPU        int    `json:"cpu"`
	Memory     int    `json:"memory"`
}

func (c *OpenIaaSVirtualMachineClient) Create(ctx context.Context, req *CreateOpenIaasVirtualMachineRequest) (string, error) {
	r := c.c.newRequest("POST", "/compute/v1/open_iaas/virtual_machines")
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
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

type UpdateOpenIaasVirtualMachineRequest struct {
	Name              string `json:"name"`
	CPU               int    `json:"cpu"`
	NumCoresPerSocket int    `json:"numCoresPerSocket"`
	Memory            int    `json:"memory"`
	SecureBoot        bool   `json:"secureBoot"`
	AutoPowerOn       bool   `json:"autoPowerOn"`
	HighAvailability  string `json:"highAvailability"`
}

func (v *OpenIaaSVirtualMachineClient) Update(ctx context.Context, id string, req *UpdateOpenIaasVirtualMachineRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/open_iaas/virtual_machines/%s", id)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *OpenIaaSVirtualMachineClient) Delete(ctx context.Context, id string) (string, error) {
	r := v.c.newRequest("DELETE", "/compute/v1/open_iaas/virtual_machines/%s", id)
	return v.c.doRequestAndReturnActivity(ctx, r)
}

type UpdateOpenIaasVirtualMachinePowerRequest struct {
	PowerState              string `json:"powerState"`
	HostId                  string `json:"hostId,omitempty"`
	Force                   bool   `json:"force,omitempty"`
	BypassMacAddressesCheck bool   `json:"bypassMacAddressesCheck,omitempty"`
	ForceShutdownDelay      int    `json:"forceShutdownDelay,omitempty"`
	BypassBlockedOperation  bool   `json:"bypassBlockedOperation,omitempty"`
}

func (v *OpenIaaSVirtualMachineClient) Power(ctx context.Context, id string, req *UpdateOpenIaasVirtualMachinePowerRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/open_iaas/virtual_machines/%s/power", id)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *OpenIaaSVirtualMachineClient) MountISO(ctx context.Context, id string, virtualDiskId string) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/open_iaas/virtual_machines/%s/mount", id)
	r.obj = map[string]string{"virtualDiskId": virtualDiskId}
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *OpenIaaSVirtualMachineClient) UnmountISO(ctx context.Context, id string) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/open_iaas/virtual_machines/%s/unmount", id)
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *OpenIaaSVirtualMachineClient) UpdateBootOrder(ctx context.Context, id string, bootOrder []string) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/open_iaas/virtual_machines/%s/boot_order", id)
	r.obj = map[string][]string{"order": bootOrder}
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (c *OpenIaaSVirtualMachineClient) WaitForTools(ctx context.Context, id string, options *WaiterOptions) (*OpenIaaSVirtualMachine, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithMaxRetries(30, b)

	var res *OpenIaaSVirtualMachine
	var count int

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		count++

		if ctx.Err() != nil {
			vm, err := c.Read(ctx, id)
			if err != nil {
				return err
			}
			res = vm
			options.log(fmt.Sprintf("timeout reached, continuing without tools for virtual machine %q", id))
			return nil
		}

		vm, err := c.Read(ctx, id)
		if err != nil {
			return options.retryableError(&StatusError{
				Code: 500,
				Body: "an error occurred while getting the status of the virtual machine"})
		}
		if vm == nil {
			err := &StatusError{
				Code: 404,
				Body: fmt.Sprintf("the virtual machine %q could not be found", id),
			}
			if count == 1 {
				return options.retryableError(err)
			}
			return options.error(err)
		}

		res = vm
		if vm.Tools.Detected {
			options.log(fmt.Sprintf("the virtual machine %q has the tools installed", id))
			return nil
		}

		return options.retryableError(&StatusError{
			Code: 500,
			Body: fmt.Sprintf("the tools are not detected for the virtual machine %q", id),
		})
	})

	// Si c'est une erreur de timeout, on l'ignore
	if err != nil && ctx.Err() != nil {
		return res, nil
	}

	return res, err
}
