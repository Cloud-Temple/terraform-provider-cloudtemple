package client

import "context"

type OpenIaasTemplateClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) Template() *OpenIaasTemplateClient {
	return &OpenIaasTemplateClient{c.c.c}
}

type TemplateDisk struct {
	ID                string
	Name              string
	Description       string
	Size              int
	StorageRepository BaseObject
}

type TemplateNetworkAdapter struct {
	Name       string
	MacAddress string
	MTU        int
	Attached   bool
	Network    BaseObject
}

type OpenIaasTemplate struct {
	ID                string
	MachineManager    BaseObject
	InternalID        string
	Name              string
	CPU               int
	NumCoresPerSocket int
	Memory            int
	PowerState        string
	Snapshots         []string
	SLAPolicies       []string
	Disks             []TemplateDisk
	NetworkAdapters   []TemplateNetworkAdapter
}

type OpenIaaSTemplateFilter struct {
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
