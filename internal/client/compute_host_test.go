package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	HostId    = "TEST_HOST_ID"
	HostName  = "TEST_HOST_NAME"
	HostMoRef = "TEST_HOST_MOREF"
)

func TestCompute_HostList(t *testing.T) {
	ctx := context.Background()
	hosts, err := client.Compute().Host().List(ctx, "", "", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(hosts), 1)

	var found bool
	for _, h := range hosts {
		if h.ID == os.Getenv(HostId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_HostRead(t *testing.T) {
	ctx := context.Background()
	host, err := client.Compute().Host().Read(ctx, os.Getenv(HostId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(HostId), host.ID)
	require.Equal(t, os.Getenv(HostName), host.Name)
	require.Equal(t, os.Getenv(HostMoRef), host.Moref)
}
