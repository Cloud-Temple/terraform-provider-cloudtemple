package client

type BackupClient struct {
	c *Client
}

func (c *Client) Backup() *BackupClient {
	return &BackupClient{c}
}
