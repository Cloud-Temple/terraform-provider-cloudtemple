package client

// PublicCloudVMClient is the sub-client for the "Public Cloud VM Instances"
// product (gateway service prefix /vm_instances, permission namespace
// public_cloud_vm_instances_*). It carries the shared *Client and therefore
// reuses the existing auth (Bearer JWT/IAM), HTTP transport, retry and activity
// polling machinery unchanged — no new auth/HTTP/polling code is introduced
// here (E0-2 / #411).
type PublicCloudVMClient struct {
	c *Client
}

// PublicCloudVM returns the Public Cloud VM Instances sub-client.
func (c *Client) PublicCloudVM() *PublicCloudVMClient {
	return &PublicCloudVMClient{c}
}
