package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	StorageId = "BACKUP_STORAGE_ID"
)

func TestBackupStorageClient_List(t *testing.T) {
	ctx := context.Background()
	storages, err := client.Backup().Storage().List(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(storages), 1)

	var found bool
	for _, st := range storages {
		if st.ID == os.Getenv(StorageId) {
			found = true
			break
		}
	}
	require.True(t, found)
}
