package client

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
)

type PATClient struct {
	c *Client
}

func (i *IAM) PAT() *PATClient {
	return &PATClient{i.c}
}

type Token struct {
	ID             string   `terraform:"id"`
	Name           string   `terraform:"name"`
	Secret         string   `terraform:"secret"`
	Roles          []string `terraform:"roles"`
	ExpirationDate string   `terraform:"expiration_date"`
}

func (p *PATClient) List(ctx context.Context, userId string, tenantId string) ([]*Token, error) {
	r := p.c.newRequest("GET", "/api/iam/v2/personal_access_tokens")
	r.params.Set("userId", userId)
	r.params.Set("tenantId", tenantId)
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Token
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (p *PATClient) Read(ctx context.Context, patID string) (*Token, error) {
	r := p.c.newRequest("GET", "/api/iam/v2/personal_access_tokens/"+patID)
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 500)
	if err != nil || !found {
		return nil, err
	}

	var out Token
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (p *PATClient) Create(ctx context.Context, name string, roles []string, expirationDate int) (*Token, error) {
	if len(roles) == 0 {
		return nil, fmt.Errorf("roles must not be empty")
	}

	r := p.c.newRequest("POST", "/api/iam/v2/personal_access_tokens")
	r.obj = map[string]interface{}{
		"name":           name,
		"roles":          roles,
		"expirationDate": expirationDate,
	}
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireHttpCodes(resp, 201); err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := decodeBody(resp, &data); err != nil {
		return nil, err
	}

	// We have to fix the type of expirationDate when creating a token
	if expirationDate, ok := data["expirationDate"].(float64); ok {
		data["expirationDate"] = fmt.Sprintf("%f", expirationDate)
	}

	var out Token
	if err := mapstructure.Decode(data, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (p *PATClient) Delete(ctx context.Context, patID string) error {
	r := p.c.newRequest("DELETE", "/api/iam/v2/personal_access_tokens/"+patID)
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return err
	}
	defer closeResponseBody(resp)
	return requireOK(resp)
}
