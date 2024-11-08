package client

type BackupOpenIaasClient struct {
	c *BackupClient
}

func (c *BackupClient) OpenIaaS() *BackupOpenIaasClient {
	return &BackupOpenIaasClient{c}
}
