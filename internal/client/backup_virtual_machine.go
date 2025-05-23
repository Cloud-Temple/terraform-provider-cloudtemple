package client

import (
	"context"
	"fmt"
	"time"

	"github.com/sethvargo/go-retry"
)

type BackupVirtualMachineClient struct {
	c *Client
}

func (c *BackupClient) VirtualMachine() *BackupVirtualMachineClient {
	return &BackupVirtualMachineClient{c.c}
}

type BackupVirtualMachineVolume struct {
	Name         string
	Key          string
	Size         int
	ConfigVolume bool
}

type BackupVirtualMachine struct {
	ID                string
	Name              string
	Moref             string
	InternalId        string
	InternalVCenterId int
	VCenterId         string
	Href              string
	MetadataPath      string
	StorageProfiles   []string
	DatacenterName    string
	Volumes           []BackupVirtualMachineVolume
}

func (c *BackupVirtualMachineClient) Read(ctx context.Context, id string) (*BackupVirtualMachine, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/virtual_machines/%s", id)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil {
		return nil, err
	} else if !found {
		return nil, nil
	}

	var out BackupVirtualMachine
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type BackupVirtualMachineNotFoundError struct {
	message        string
	virtualMachine string
}

const backupVirtualMachineNotFoundErrorMessage = `
  Message: %s
  VirtualMachine: %s
`

func (b *BackupVirtualMachineNotFoundError) Error() string {
	if b.virtualMachine == "" {
		return b.message
	}

	return fmt.Sprintf(
		backupVirtualMachineNotFoundErrorMessage,
		b.message,
		b.virtualMachine,
	)
}

func (c *BackupVirtualMachineClient) WaitForInventory(ctx context.Context, id string, options *WaiterOptions) (*BackupVirtualMachine, error) {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithCappedDuration(30*time.Second, b)

	var res *BackupVirtualMachine
	var count int

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		count++
		virtualMachine, err := c.Read(ctx, id)
		if err != nil {
			return options.error(err)
		}
		if virtualMachine == nil {
			err := &BackupVirtualMachineNotFoundError{
				message:        fmt.Sprintf("the virtual machine %q could not be found", id),
				virtualMachine: id,
			}
			return options.retryableError(err)
		}
		res = virtualMachine
		options.log(fmt.Sprintf("the virtual machine %q has been found.", id))
		return nil
	})
	return res, err

}
