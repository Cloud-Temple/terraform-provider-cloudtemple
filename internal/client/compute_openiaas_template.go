package client

import "context"

type OpenIaasTemplateClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) Template() *OpenIaasTemplateClient {
	return &OpenIaasTemplateClient{c.c.c}
}

type TemplateDisk struct {
	Name              string     `terraform:"name"`
	Description       string     `terraform:"description"`
	Size              int        `terraform:"size"`
	StorageRepository BaseObject `terraform:"storage_repository"`
}

type TemplateNetworkAdapter struct {
	Name       string     `terraform:"name"`
	MacAddress string     `terraform:"mac_address"`
	MTU        int        `terraform:"mtu"`
	Attached   bool       `terraform:"attached"`
	Network    BaseObject `terraform:"network"`
}

type OpenIaasTemplate struct {
	ID             string `terraform:"id"`
	MachineManager struct {
		ID   string `terraform:"id"`
		Name string `terraform:"name"`
		Type string `terraform:"type"`
	} `terraform:"machine_manager"`
	InternalID        string                   `terraform:"internal_id"`
	Name              string                   `terraform:"name"`
	CPU               int                      `terraform:"cpu"`
	NumCoresPerSocket int                      `terraform:"num_cores_per_socket"`
	Memory            int                      `terraform:"memory"`
	PowerState        string                   `terraform:"power_state"`
	Snapshots         []string                 `terraform:"snapshots"`
	SLAPolicies       []string                 `terraform:"sla_policies"`
	Disks             []TemplateDisk           `terraform:"disks"`
	NetworkAdapters   []TemplateNetworkAdapter `terraform:"network_adapters"`
}

type OpenIaaSTemplateFilter struct {
	// TODO : Add filter by name
	MachineManagerId string `filter:"machineManagerId"`
	PoolId           string `filter:"poolId,omitempty"`
}

func (p *OpenIaasTemplateClient) List(
	ctx context.Context,
	filter *OpenIaaSTemplateFilter) ([]*OpenIaasTemplate, error) {

	r := p.c.newRequest("GET", "/compute/v1/open_iaas/templates")
	r.addFilter(filter)
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*OpenIaasTemplate
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (p *OpenIaasTemplateClient) Read(ctx context.Context, id string) (*OpenIaasTemplate, error) {
	r := p.c.newRequest("GET", "/compute/v1/open_iaas/templates/%s", id)
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaasTemplate
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
