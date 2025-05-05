package client

import (
	"context"
	"fmt"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	HostId    = "COMPUTE_HOST_ID"
	HostName  = "COMPUTE_HOST_NAME"
	HostMoRef = "COMPUTE_HOST_MOREF"
)

func TestCompute_HostList(t *testing.T) {
	ctx := context.Background()
	hosts, err := client.Compute().Host().List(ctx, &clientpkg.HostFilter{})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(hosts), 1)

	var found bool
	for _, h := range hosts {
		if h.ID == os.Getenv(HostId) {
			fmt.Println("found")
			found = true
			break
		}
	}

	require.True(t, found)
}

func TestCompute_HostRead(t *testing.T) {
	ctx := context.Background()
	fmt.Println(os.Getenv(HostId))
	host, err := client.Compute().Host().Read(ctx, os.Getenv(HostId))

	require.NoError(t, err)

	require.Equal(t, os.Getenv(HostId), host.ID)
	require.Equal(t, os.Getenv(HostName), host.Name)
	require.Equal(t, os.Getenv(HostMoRef), host.Moref)
}
