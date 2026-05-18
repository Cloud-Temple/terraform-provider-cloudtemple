package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	StaticIPId = "STATIC_IP_ID"
)

func TestVPC_StaticIPList(t *testing.T) {
	ctx := context.Background()
	networkID := os.Getenv(PrivateNetworkId)
	if networkID == "" {
		t.Skip("PRIVATE_NETWORK_ID not set, skipping test")
	}

	staticIPs, err := client.VPC().StaticIP().List(ctx, networkID, &clientpkg.StaticIPFilter{})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(staticIPs), 0)
}

func TestVPC_StaticIPRead(t *testing.T) {
	ctx := context.Background()
	staticIPID := os.Getenv(StaticIPId)
	if staticIPID == "" {
		t.Skip("STATIC_IP_ID not set, skipping test")
	}

	staticIP, err := client.VPC().StaticIP().Read(ctx, staticIPID)
	require.NoError(t, err)
	require.NotNil(t, staticIP)

	require.Equal(t, staticIPID, staticIP.ID)
	require.NotEmpty(t, staticIP.IPAddress)
	require.NotEmpty(t, staticIP.MacAddress)
}
