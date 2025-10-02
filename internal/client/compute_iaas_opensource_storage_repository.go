package client

import "context"

type OpenIaaSStorageRepositoryClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) StorageRepository() *OpenIaaSStorageRepositoryClient {
	return &OpenIaaSStorageRepositoryClient{c.c.c}
}

type OpenIaaSStorageRepository struct {
	ID              string
	InternalId      string
	Name            string
	Description     string
	MaintenanceMode bool
	MaxCapacity     int
	FreeCapacity    int
	StorageType     string
	VirtualDisks    []string
	Shared          bool
	Accessible      int
	Host            BaseObject
	Pool            BaseObject
	MachineManager  BaseObject
}

type StorageRepositoryFilter struct {
	MachineManagerId string   `filter:"machineManagerId"`
	PoolId           string   `filter:"poolId"`
	HostId           string   `filter:"hostId"`
	StorageTypes     []string `filter:"types,omitempty"`
	Shared           bool     `filter:"shared"`
}

func (h *OpenIaaSStorageRepositoryClient) List(
	ctx context.Context,
	filter *StorageRepositoryFilter) ([]*OpenIaaSStorageRepository, error) {

	r := h.c.newRequest("GET", "/compute/v1/open_iaas/storage_repositories")
	r.addFilter(filter)
	resp, err := h.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*OpenIaaSStorageRepository
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (h *OpenIaaSStorageRepositoryClient) Read(ctx context.Context, id string) (*OpenIaaSStorageRepository, error) {
	r := h.c.newRequest("GET", "/compute/v1/open_iaas/storage_repositories/%s", id)
	resp, err := h.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaaSStorageRepository
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
