package client

import (
	"context"
)

type BackupVirtualMachineClient struct {
	c *Client
}

func (c *BackupClient) VirtualMachine() *BackupVirtualMachineClient {
	return &BackupVirtualMachineClient{c.c}
}

type BackupVirtualMachineVolume struct {
	Name         string `terraform:"name"`
	Key          int    `terraform:"key"`
	Size         int    `terraform:"size"`
	ConfigVolume bool   `terraform:"config_volume"`
}

type BackupVirtualMachine struct {
	ID                string                       `terraform:"id"`
	Name              string                       `terraform:"name"`
	Moref             string                       `terraform:"moref"`
	InternalId        string                       `terraform:"internal_id"`
	InternalVCenterId string                       `terraform:"internal_vcenter_id"`
	VCenterId         string                       `terraform:"vcenter_id"`
	Href              string                       `terraform:"href"`
	MetadataPath      string                       `terraform:"matadata_path"`
	StorageProfiles   []string                     `terraform:"storage_profiles"`
	DatacenterName    string                       `terraform:"datacenter_name"`
	Volumes           []BackupVirtualMachineVolume `terraform:"volumes"`
}

func (c *BackupVirtualMachineClient) Read(ctx context.Context, id string) (*BackupVirtualMachine, error) {
	r := c.c.newRequest("GET", "/api/backup/v1/virtual_machines/%s", id)
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
