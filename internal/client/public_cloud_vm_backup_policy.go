package client

import "context"

type PublicCloudVMBackupPolicyClient struct {
	c *Client
}

// BackupPolicy returns the backup-policy catalogue sub-client (read-only,
// list-only: the API returns the whole tenant policy list with no by-id endpoint
// or server-side filter). A backup policy is a mandatory input of the VM resource
// (backup_policy_id is required at VM create).
func (v *PublicCloudVMClient) BackupPolicy() *PublicCloudVMBackupPolicyClient {
	return &PublicCloudVMBackupPolicyClient{v.c}
}

// PublicCloudVMBackupPolicy mirrors an element of GET /vm_instances/v1/backup_policies
// (bare array, camelCase). Several fields are documented as nullable and decode to
// their zero value when null.
type PublicCloudVMBackupPolicy struct {
	ID                            string
	Name                          string
	Description                   string
	Retention                     int
	ScheduleCron                  string
	ScheduleWindowStartHour       int
	ScheduleWindowDurationMinutes int
}

// List returns the tenant's backup policies (bare JSON array, no server-side
// filter).
func (b *PublicCloudVMBackupPolicyClient) List(ctx context.Context) ([]*PublicCloudVMBackupPolicy, error) {
	req := b.c.newRequest("GET", "/vm_instances/v1/backup_policies")
	resp, err := b.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*PublicCloudVMBackupPolicy
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
