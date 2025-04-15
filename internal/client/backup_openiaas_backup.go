package client

import "context"

type BackupOpenIaasBackupClient struct {
	c *Client
}

func (c *BackupOpenIaasClient) Backup() *BackupOpenIaasBackupClient {
	return &BackupOpenIaasBackupClient{c.c.c}
}

type Backup struct {
	ID                      string
	InternalID              string
	Mode                    string
	IsVirtualMachineDeleted bool
	Size                    int
	Timestamp               int
	VirtualMachine          BaseObject
	Policy                  BaseObject
}

func (v *BackupOpenIaasBackupClient) Read(ctx context.Context, id string) (*Backup, error) {
	r := v.c.newRequest("GET", "/backup/v1/open_iaas/backups/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out Backup
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type OpenIaasBackupFilter struct {
	MachineManagerId string `filter:"machineManagerId"`
	VirtualMachineId string `filter:"virtualMachineId"`
	Deleted          bool   `filter:"deleted"`
}

func (v *BackupOpenIaasBackupClient) List(ctx context.Context, filter *OpenIaasBackupFilter) ([]*Backup, error) {
	r := v.c.newRequest("GET", "/backup/v1/open_iaas/backups")
	r.addFilter(filter)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out []*Backup
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
