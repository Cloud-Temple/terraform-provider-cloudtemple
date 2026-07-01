package client

import "context"

type PublicCloudVMStorageTypeClient struct {
	c *Client
}

// StorageType returns the storage-type catalogue sub-client (read-only, list-only:
// the API has no by-id endpoint and no filter for storage types).
func (v *PublicCloudVMClient) StorageType() *PublicCloudVMStorageTypeClient {
	return &PublicCloudVMStorageTypeClient{v.c}
}

// PublicCloudVMStorageType mirrors an element of GET /vm_instances/v1/storage_types.
// The endpoint returns a WRAPPED object { "storageTypes": [...] }.
type PublicCloudVMStorageType struct {
	ID          string
	Name        string
	Description string
	IopsHint    string
	MinSizeGb   int
	MaxSizeGb   int
	IsAvailable bool
}

type publicCloudVMStorageTypeListResponse struct {
	StorageTypes []*PublicCloudVMStorageType
}

// List returns the storage types available to the tenant (wrapped response, no
// server-side filter).
func (s *PublicCloudVMStorageTypeClient) List(ctx context.Context) ([]*PublicCloudVMStorageType, error) {
	req := s.c.newRequest("GET", "/vm_instances/v1/storage_types")
	resp, err := s.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out publicCloudVMStorageTypeListResponse
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out.StorageTypes, nil
}
