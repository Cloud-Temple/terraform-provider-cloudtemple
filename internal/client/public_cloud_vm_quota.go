package client

import (
	"context"
	"encoding/json"
)

type PublicCloudVMQuotaClient struct {
	c *Client
}

// Quota returns the quota sub-client (read-only, singleton scoped to the JWT
// tenant).
func (v *PublicCloudVMClient) Quota() *PublicCloudVMQuotaClient {
	return &PublicCloudVMQuotaClient{v.c}
}

// PublicCloudVMQuota mirrors GET /vm_instances/v1/quotas (a single object).
// Units are strict and binary: RAM in MiB, storage in GiB, vCPU in units.
type PublicCloudVMQuota struct {
	VcpuLimit       int
	RamLimitMib     int
	StorageLimitGib int
	VcpuUsed        int
	RamUsedMib      int
	StorageUsedGib  int
}

// UnmarshalJSON accepts both the current `*Mib`/`*Gib` spellings and the
// deprecated `*Mb`/`*Gb` ones (see public_cloud_vm_capacity.go, issue #524).
// The two LIMIT fields are required by the API contract, so their absence under
// both spellings is an error. The two USAGE counters carry `default: 0` in the
// contract instead, so an absence there legitimately means zero.
func (q *PublicCloudVMQuota) UnmarshalJSON(data []byte) error {
	type plain PublicCloudVMQuota
	var v struct {
		plain
		RamLimitMib     *int
		RamLimitMb      *int
		StorageLimitGib *int
		StorageLimitGb  *int
		RamUsedMib      *int
		RamUsedMb       *int
		StorageUsedGib  *int
		StorageUsedGb   *int
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*q = PublicCloudVMQuota(v.plain)

	const subject = "the tenant quota"
	ramLimit, err := resolveRenamedCapacity(v.RamLimitMib, v.RamLimitMb, "ramLimitMib", "ramLimitMb", subject)
	if err != nil {
		return err
	}
	storageLimit, err := resolveRenamedCapacity(v.StorageLimitGib, v.StorageLimitGb, "storageLimitGib", "storageLimitGb", subject)
	if err != nil {
		return err
	}
	ramUsed, err := resolveOptionalRenamedCapacity(v.RamUsedMib, v.RamUsedMb, "ramUsedMib", "ramUsedMb", subject)
	if err != nil {
		return err
	}
	storageUsed, err := resolveOptionalRenamedCapacity(v.StorageUsedGib, v.StorageUsedGb, "storageUsedGib", "storageUsedGb", subject)
	if err != nil {
		return err
	}

	q.RamLimitMib = ramLimit
	q.StorageLimitGib = storageLimit
	q.RamUsedMib = ramUsed
	q.StorageUsedGib = storageUsed
	return nil
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
