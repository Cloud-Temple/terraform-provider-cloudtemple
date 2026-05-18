package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	FloatingIPId = "FLOATING_IP_ID"
)

func TestVPC_FloatingIPList(t *testing.T) {
	ctx := context.Background()
	floatingIPs, err := client.VPC().FloatingIP().List(ctx, &clientpkg.FloatingIPFilter{})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(floatingIPs), 0)
}

func TestVPC_FloatingIPListFiltered(t *testing.T) {
	ctx := context.Background()
	vpcID := os.Getenv(VPCId)
	if vpcID == "" {
		t.Skip("VPC_ID not set, skipping test")
	}

	floatingIPs, err := client.VPC().FloatingIP().List(ctx, &clientpkg.FloatingIPFilter{
		VpcID: vpcID,
	})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(floatingIPs), 0)
}

func TestVPC_FloatingIPRead(t *testing.T) {
	ctx := context.Background()
	floatingIPID := os.Getenv(FloatingIPId)
	if floatingIPID == "" {
		t.Skip("FLOATING_IP_ID not set, skipping test")
	}

	floatingIP, err := client.VPC().FloatingIP().Read(ctx, floatingIPID)
	require.NoError(t, err)
	require.NotNil(t, floatingIP)

	require.Equal(t, floatingIPID, floatingIP.ID)
	require.NotEmpty(t, floatingIP.IPAddress)
}
