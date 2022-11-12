package client

type BackupBackupClient struct {
	c *Client
}

func (c *BackupClient) Backup() *BackupBackupClient {
	return &BackupBackupClient{c.c}
}
