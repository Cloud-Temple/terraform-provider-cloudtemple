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
	// Sku is the priced SKU of the storage resource. It is a pointer because
	// the API may omit it (or send null) for a given storage type; a nil Sku
	// then flattens to an empty list rather than a phantom zero-priced object.
	Sku *PublicCloudVMSku
}

// PublicCloudVMSku is the priced SKU the VM Instances API carries on each
// storage type. Unlike its sibling fields, this struct pins EXPLICIT json tags:
// they lock the exact API spelling of these billing fields (notably the
// French/English description pair) instead of relying on case-insensitive
// matching.
type PublicCloudVMSku struct {
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Unit          string  `json:"unit"`
	Description   string  `json:"description"`
	DescriptionEn string  `json:"descriptionEn"`
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
