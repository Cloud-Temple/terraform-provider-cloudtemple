package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupStorageClient_List(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	storages, err := client.Backup().Storage().List(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(storages), 1)

	var found bool
	for _, st := range storages {
		if st.ID == "2101" {
			found = true
			break
		}
	}
	require.True(t, found)
}
