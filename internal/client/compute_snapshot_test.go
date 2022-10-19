package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_SnapshotList(t *testing.T) {
	ctx := context.Background()
	_, err := client.Compute().Snapshot().List(ctx, "de2b8b80-8b90-414a-bc33-e12f61a4c05c")
	require.NoError(t, err)
}
