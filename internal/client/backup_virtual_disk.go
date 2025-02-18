package client

import (
	"context"
	"fmt"
	"time"

	"github.com/sethvargo/go-retry"
)

type BackupVirtualDiskClient struct {
	c *Client
}

func (c *BackupClient) VirtualDisk() *BackupVirtualDiskClient {
	return &BackupVirtualDiskClient{c.c}
}

type BackupVirtualDisk struct {
	ID               string `terraform:"id"`
	Name             string `terraform:"name"`
	InternalId       string `terraform:"internal_id"`
	InstanceId       string `terraform:"instance_id"`
	SPPServerId      string `terraform:"spp_server_id"`
	VirtualMachineId string `terraform:"virtual_machine_id"`
}

type BackupVirtualDiskNotFoundError struct {
	message     string
	virtualDisk string
}

const backupVirtualDiskNotFoundErrorMessage = `
  Message: %s
  VirtualDisk: %s
`

func (b *BackupVirtualDiskNotFoundError) Error() string {
	if b.virtualDisk == "" {
		return b.message
	}

	return fmt.Sprintf(
		backupVirtualDiskNotFoundErrorMessage,
		b.message,
		b.virtualDisk,
	)
}

func (c *BackupVirtualDiskClient) Read(ctx context.Context, id string) (*BackupVirtualDisk, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/virtual_disks/%s", id)
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

	var out BackupVirtualDisk
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *BackupVirtualDiskClient) WaitForInventory(ctx context.Context, id string, options *WaiterOptions) (*BackupVirtualDisk, error) {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithCappedDuration(30*time.Second, b)

	var res *BackupVirtualDisk
	var count int

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		count++
		virtualDisk, err := c.Read(ctx, id)
		if err != nil {
			return options.error(err)
		}
		if virtualDisk == nil {
			err := &BackupVirtualDiskNotFoundError{
				message:     fmt.Sprintf("the virtual disk %q could not be found", id),
				virtualDisk: id,
			}
			return options.retryableError(err)
		}
		res = virtualDisk
		options.log(fmt.Sprintf("the virtual disk %q has been found.", id))
		return nil
	})
	return res, err

}
