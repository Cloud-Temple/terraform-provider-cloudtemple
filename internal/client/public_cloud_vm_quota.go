package client

import "context"

type PublicCloudVMQuotaClient struct {
	c *Client
}

// Quota returns the quota sub-client (read-only, singleton scoped to the JWT
// tenant).
func (v *PublicCloudVMClient) Quota() *PublicCloudVMQuotaClient {
	return &PublicCloudVMQuotaClient{v.c}
}

// PublicCloudVMQuota mirrors GET /vm_instances/v1/quotas (a single object).
// Units are strict: RAM in MB, storage in GB, vCPU in units.
type PublicCloudVMQuota struct {
	VcpuLimit      int
	RamLimitMb     int
	StorageLimitGb int
	VcpuUsed       int
	RamUsedMb      int
	StorageUsedGb  int
}

// Read returns the tenant quota. Note: a 404 here means "no worker found for the
// tenant" (a configuration error), NOT "no quota" — so it is surfaced as an error
// (requireOK), never as an absence.
func (q *PublicCloudVMQuotaClient) Read(ctx context.Context) (*PublicCloudVMQuota, error) {
	req := q.c.newRequest("GET", "/vm_instances/v1/quotas")
	resp, err := q.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out PublicCloudVMQuota
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
