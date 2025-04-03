package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	SnapShotId = "COMPUTE_SNAPSHOT_ID"
)

func TestCompute_SnapshotList(t *testing.T) {
	ctx := context.Background()
	_, err := client.Compute().Snapshot().List(ctx, os.Getenv(SnapShotId))
	require.NoError(t, err)
}
