package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupStorageClient_List(t *testing.T) {
	ctx := context.Background()
	storages, err := client.Backup().Storage().List(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(storages), 1)
}
