package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type PublicCloudVMStorageTypeClient struct {
	c *Client
}

// StorageType returns the storage-type catalogue sub-client (read-only, list-only:
// the API has no by-id endpoint for storage types).
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
	MinSizeGib  int
	MaxSizeGib  int
	IsAvailable bool
	// Sku is the priced SKU of the storage resource. It is a pointer because
	// the API may omit it (or send null) for a given storage type; a nil Sku
	// then flattens to an empty list rather than a phantom zero-priced object.
	Sku *PublicCloudVMSku
}

// UnmarshalJSON accepts both the current `minSizeGib`/`maxSizeGib` spellings and
// the deprecated `minSizeGb`/`maxSizeGb` ones (see public_cloud_vm_capacity.go,
// issue #524).
func (s *PublicCloudVMStorageType) UnmarshalJSON(data []byte) error {
	type plain PublicCloudVMStorageType
	var v struct {
		plain
		MinSizeGib *int
		MinSizeGb  *int
		MaxSizeGib *int
		MaxSizeGb  *int
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*s = PublicCloudVMStorageType(v.plain)

	subject := "storage type " + quotedOrUnidentified(v.plain.ID)
	minSize, err := resolveRenamedCapacity(v.MinSizeGib, v.MinSizeGb, "minSizeGib", "minSizeGb", subject)
	if err != nil {
		return err
	}
	maxSize, err := resolveRenamedCapacity(v.MaxSizeGib, v.MaxSizeGb, "maxSizeGib", "maxSizeGb", subject)
	if err != nil {
		return err
	}
	s.MinSizeGib = minSize
	s.MaxSizeGib = maxSize
	return nil
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

// PublicCloudVMStorageTypeFilter carries the optional list filters. The two
// fields are optional but PAIRED: the storage-types API answers an opaque 500
// to a request carrying only one of them (live-probed), so List refuses a lone
// filter up-front with an actionable error instead of letting it reach the wire.
type PublicCloudVMStorageTypeFilter struct {
	AvailabilityZoneID string `filter:"availabilityZoneId"`
	InstanceFamilyID   string `filter:"instanceFamilyId"`
}

// List returns the storage types available to the tenant (wrapped response),
// optionally filtered by an availability-zone/instance-family pair. filter may
// be nil; when non-nil, its two fields must be both set or both empty.
func (s *PublicCloudVMStorageTypeClient) List(ctx context.Context, filter *PublicCloudVMStorageTypeFilter) ([]*PublicCloudVMStorageType, error) {
	if filter != nil && (filter.AvailabilityZoneID == "") != (filter.InstanceFamilyID == "") {
		return nil, fmt.Errorf("storage-type filters availabilityZoneId and instanceFamilyId must be used together (got availabilityZoneId=%q, instanceFamilyId=%q)", filter.AvailabilityZoneID, filter.InstanceFamilyID)
	}
	req := s.c.newRequest("GET", "/vm_instances/v1/storage_types")
	req.addFilter(filter)
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
