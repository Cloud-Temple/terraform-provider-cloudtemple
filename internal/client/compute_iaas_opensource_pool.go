package client

import "context"

type OpenIaasPoolClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) Pool() *OpenIaasPoolClient {
	return &OpenIaasPoolClient{c.c.c}
}

type OpenIaasPool struct {
	ID                      string
	MachineManager          BaseObject
	InternalID              string
	Name                    string
	Label                   string
	HighAvailabilityEnabled bool
	Master                  string
	Hosts                   []string
	Memory                  struct {
		Usage int
		Size  int
	}
	Cpu struct {
		Cores   int
		Sockets int
	}
	Type struct {
		Key         string
		Description string
	}
}

type OpenIaasPoolFilter struct {
	MachineManagerId string `filter:"machineManagerId"`
}

func (p *OpenIaasPoolClient) List(
	ctx context.Context,
	filter *OpenIaasPoolFilter) ([]*OpenIaasPool, error) {

	r := p.c.newRequest("GET", "/compute/v1/open_iaas/pools")
	r.addFilter(filter)
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*OpenIaasPool
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (p *OpenIaasPoolClient) Read(ctx context.Context, id string) (*OpenIaasPool, error) {
	r := p.c.newRequest("GET", "/compute/v1/open_iaas/pools/%s", id)
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaasPool
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
