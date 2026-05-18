package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	PrivateNetworkId = "PRIVATE_NETWORK_ID"
)

func TestVPC_PrivateNetworkList(t *testing.T) {
	ctx := context.Background()
	networks, err := client.VPC().PrivateNetwork().List(ctx, &clientpkg.PrivateNetworkFilter{})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(networks), 0)
}

func TestVPC_PrivateNetworkListFiltered(t *testing.T) {
	ctx := context.Background()
	vpcID := os.Getenv(VPCId)
	if vpcID == "" {
		t.Skip("VPC_ID not set, skipping test")
	}

	networks, err := client.VPC().PrivateNetwork().List(ctx, &clientpkg.PrivateNetworkFilter{
		VpcID: vpcID,
	})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(networks), 0)
}

func TestVPC_PrivateNetworkRead(t *testing.T) {
	ctx := context.Background()
	networkID := os.Getenv(PrivateNetworkId)
	if networkID == "" {
		t.Skip("PRIVATE_NETWORK_ID not set, skipping test")
	}

	network, err := client.VPC().PrivateNetwork().Read(ctx, networkID)
	require.NoError(t, err)
	require.NotNil(t, network)

	require.Equal(t, networkID, network.ID)
	require.NotEmpty(t, network.IPAddress)
}
