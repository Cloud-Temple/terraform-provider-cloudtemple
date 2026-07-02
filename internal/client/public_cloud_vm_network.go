package client

import "context"

type PublicCloudVMNetworkClient struct {
	c *Client
}

// Network returns the VM network catalogue sub-client. Networks are a read-only
// catalogue provisioned upstream (Saturne/VPC): the provider never creates or
// deletes them, and both endpoints (GET /networks, GET /networks/{id}) are
// synchronous — there is no activity to poll.
func (v *PublicCloudVMClient) Network() *PublicCloudVMNetworkClient {
	return &PublicCloudVMNetworkClient{v.c}
}

// PublicCloudVMNetwork mirrors an element of GET /networks (bare JSON array,
// verified live). VPC-backed networks carry a `vpc` block (verified live since
// 2026-07); the key is omitted entirely on Private Backbone networks, so VPC is
// nil there — which is how a caller tells the two kinds apart.
type PublicCloudVMNetwork struct {
	ID   string
	Name string
	VPC  *PublicCloudVMNetworkVPC
}

// PublicCloudVMNetworkVPC is the `vpc` block of a VPC-backed network:
// {"id","name","privateNetwork":{"id","name"}}.
type PublicCloudVMNetworkVPC struct {
	ID             string
	Name           string
	PrivateNetwork *PublicCloudVMNetworkRef
}

// PublicCloudVMNetworkRef is a minimal {id,name} reference.
type PublicCloudVMNetworkRef struct {
	ID   string
	Name string
}

// List returns the network catalogue (bare JSON array, lenient success contract).
func (n *PublicCloudVMNetworkClient) List(ctx context.Context) ([]*PublicCloudVMNetwork, error) {
	return n.list(ctx, false)
}

// ListStrict returns the catalogue with a 200-only contract. The response is a
// bare array with no `total`, so there is no completeness cross-check to run —
// and a read-only catalogue is never the basis for a state-drop decision, so a
// 200-only contract is sufficient.
func (n *PublicCloudVMNetworkClient) ListStrict(ctx context.Context) ([]*PublicCloudVMNetwork, error) {
	return n.list(ctx, true)
}

func (n *PublicCloudVMNetworkClient) list(ctx context.Context, strict bool) ([]*PublicCloudVMNetwork, error) {
	req := n.c.newRequest("GET", "/vm_instances/v1/networks")
	resp, err := n.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	if strict {
		if err := requireHttpCodes(resp, 200); err != nil {
			return nil, err
		}
	} else {
		if err := requireOK(resp); err != nil {
			return nil, err
		}
	}

	var out []*PublicCloudVMNetwork
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Read returns a single network by id. Absence does NOT surface as a clean 404:
// the live platform returns 400 (well-formed unknown id) or 500 (nil UUID), so
// any non-200 fails closed with an error. That is correct for a read-only data
// source — it errors on a bad id rather than dropping anything from state.
func (n *PublicCloudVMNetworkClient) Read(ctx context.Context, id string) (*PublicCloudVMNetwork, error) {
	req := n.c.newRequest("GET", "/vm_instances/v1/networks/%s", id)
	resp, err := n.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	// 200 only: absence on this endpoint is 400/500 (never a clean 404), and a
	// read-only lookup has no reason to accept 201/206 — anything but 200 fails
	// closed with an error rather than decoding a partial/unexpected body.
	if err := requireHttpCodes(resp, 200); err != nil {
		return nil, err
	}

	var out PublicCloudVMNetwork
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
