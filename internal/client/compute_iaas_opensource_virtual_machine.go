package client

import (
	"context"
	"fmt"
	"time"
)

type OpenIaaSVirtualMachineClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) VirtualMachine() *OpenIaaSVirtualMachineClient {
	return &OpenIaaSVirtualMachineClient{c.c.c}
}

type OpenIaaSVirtualMachine struct {
	ID                  string
	Name                string
	InternalID          string
	PowerState          string
	SecureBoot          bool
	HighAvailability    string
	BootFirmware        string
	AutoPowerOn         bool
	DvdDrive            DvdDrive
	BootOrder           []string
	OperatingSystemName string
	CPU                 int
	NumCoresPerSocket   int
	Memory              int
	Tools               struct { // Deprecated, use PVDrivers and ManagementAgent instead
		Detected bool
		Version  string
	}
	PVDrivers struct {
		Detected    bool
		Version     string
		AreUpToDate bool
	}
	ManagementAgent struct {
		Detected bool
	}
	Addresses struct {
		IPv6 string
		IPv4 string
	}
	MachineManager BaseObject
	Host           BaseObject
	Pool           BaseObject
}

type DvdDrive struct {
	Name     string
	Attached bool
}

type OpenIaaSVirtualMachineFilter struct {
	MachineManagerID string `filter:"machineManagerId"`
}

type CloudInit struct {
	CloudConfig   string `json:"cloudConfig,omitempty"`
	NetworkConfig string `json:"networkConfig,omitempty"`
}

type OSNetworkAdapter struct {
	NetworkID string `json:"networkId"`
	MAC       string `json:"mac,omitempty"`
}

type CreateOpenIaasVirtualMachineRequest struct {
	Name            string             `json:"name"`
	TemplateID      string             `json:"templateId"`
	CPU             int                `json:"cpu"`
	Memory          int                `json:"memory"`
	CloudInit       CloudInit          `json:"cloudInit,omitempty"`
	NetworkAdapters []OSNetworkAdapter `json:"networkAdapters,omitempty"`
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
	Name              string `json:"name,omitempty"`
	CPU               int    `json:"cpu,omitempty"`
	NumCoresPerSocket int    `json:"numCoresPerSocket,omitempty"`
	Memory            int    `json:"memory,omitempty"`
	SecureBoot        bool   `json:"secureBoot"`
	BootFirmware      string `json:"bootFirmware,omitempty"`
	AutoPowerOn       bool   `json:"autoPowerOn"`
	HighAvailability  string `json:"highAvailability,omitempty"`
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

func (c *OpenIaaSVirtualMachineClient) WaitForDrivers(
	ctx context.Context,
	id string,
	timeout time.Duration,
	options *WaiterOptions,
) (*OpenIaaSVirtualMachine, error) {

	if timeout == 0 {
		options.log(fmt.Sprintf(
			"[WAITER] skipping wait for drivers for virtual machine %q (timeout = 0)",
			id,
		))
		return c.Read(ctx, id)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastVM *OpenIaaSVirtualMachine

	for {
		select {

		case <-timeoutCtx.Done():
			options.log(fmt.Sprintf(
				"[WAITER] timeout reached, continuing without PV drivers for virtual machine %q",
				id,
			))
			return lastVM, nil

		case <-ticker.C:

			if timeoutCtx.Err() != nil {
				options.log("[WAITER] timeout reached while waiting, continuing without PV drivers")
				return lastVM, nil
			}

			vm, err := c.Read(timeoutCtx, id)
			if err != nil {
				return nil, fmt.Errorf(
					"[WAITER] failed to read virtual machine %q while waiting for drivers: %s",
					id, err,
				)
			}

			if vm == nil {
				return nil, fmt.Errorf(
					"[WAITER] the virtual machine %q could not be found",
					id,
				)
			}

			lastVM = vm
			if vm.PVDrivers.Detected {
				options.log(fmt.Sprintf(
					"[WAITER] the virtual machine %q has the PV drivers detected",
					id,
				))
				return vm, nil
			}
		}
	}
}
