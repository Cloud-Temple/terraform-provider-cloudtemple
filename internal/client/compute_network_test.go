package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	NetworkId    = "COMPUTE_NETWORK_ID"
	NetworkName  = "COMPUTE_NETWORK_NAME"
	NetworkMoRef = "COMPUTE_NETWORK_MOREF"
)

func TestCompute_NetworkList(t *testing.T) {
	ctx := context.Background()
	networks, err := client.Compute().Network().List(ctx, &NetworkFilter{})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(networks), 1)

	var found bool
	for _, h := range networks {
		if h.ID == os.Getenv(NetworkId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_NetworkRead(t *testing.T) {
	ctx := context.Background()
	network, err := client.Compute().Network().Read(ctx, os.Getenv(NetworkId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(NetworkId), network.ID)
	require.Equal(t, os.Getenv(NetworkName), network.Name)
	require.Equal(t, os.Getenv(NetworkMoRef), network.Moref)
	require.Equal(t, os.Getenv(MachineManagerId2), network.MachineManagerId)
}
