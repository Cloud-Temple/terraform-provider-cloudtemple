package client

import "context"

type OpenIaaSStorageRepositoryClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) StorageRepository() *OpenIaaSStorageRepositoryClient {
	return &OpenIaaSStorageRepositoryClient{c.c.c}
}

type OpenIaaSStorageRepository struct {
	ID                string     `terraform:"id"`
	InternalId        string     `terraform:"internal_id"`
	Name              string     `terraform:"name"`
	Description       string     `terraform:"description"`
	MaintenanceStatus bool       `terraform:"maintenance_status"`
	MaxCapacity       int        `terraform:"max_capacity"`
	FreeCapacity      int        `terraform:"free_capacity"`
	StorageType       string     `terraform:"type"`
	VirtualDisks      []string   `terraform:"virtual_disks"`
	Shared            bool       `terraform:"shared"`
	Accessible        int        `terraform:"accessible"`
	Host              BaseObject `terraform:"host"`
	Pool              BaseObject `terraform:"pool"`
	MachineManager    struct {
		ID   string `terraform:"id"`
		Name string `terraform:"name"`
	} `terraform_flatten:"machine_manager"`
}

type StorageRepositoryFilter struct {
	// TODO : Add filter by name
	MachineManagerId string   `filter:"machineManagerId"`
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
