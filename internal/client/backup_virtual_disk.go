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
	ID               string
	Name             string
	InternalId       string
	InstanceId       string
	SPPServerId      string
	VirtualMachineId string
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

	return waitForBackupVirtualDiskInventory(ctx, id, func(ctx context.Context) (*BackupVirtualDisk, error) {
		return c.Read(ctx, id)
	}, b, options)
}

// backupVirtualDiskReadFunc abstracts the read so the inventory polling loop can
// be unit tested without HTTP calls or real sleeps.
type backupVirtualDiskReadFunc func(ctx context.Context) (*BackupVirtualDisk, error)

// waitForBackupVirtualDiskInventory is the polling loop behind WaitForInventory,
// with read and backoff injected. Behavior preserved exactly: ANY read error is
// fatal at once (no transient-vs-permanent distinction — see #293 Finding F1; NOT
// changed here), a nil result keeps polling (waiting for the disk to appear in the
// inventory), and a found disk returns. The poll is bounded only by the injected
// backoff / context.
func waitForBackupVirtualDiskInventory(ctx context.Context, id string, read backupVirtualDiskReadFunc, b retry.Backoff, options *WaiterOptions) (*BackupVirtualDisk, error) {
	var res *BackupVirtualDisk
	var count int

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		count++
		virtualDisk, err := read(ctx)
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
