package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	VPCId   = "VPC_ID"
	VPCName = "VPC_NAME"
)

func TestVPC_VPCList(t *testing.T) {
	ctx := context.Background()
	vpcs, err := client.VPC().VPC().List(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(vpcs), 0)
}

func TestVPC_VPCRead(t *testing.T) {
	ctx := context.Background()
	vpcID := os.Getenv(VPCId)
	if vpcID == "" {
		t.Skip("VPC_ID not set, skipping test")
	}

	vpc, err := client.VPC().VPC().Read(ctx, vpcID)
	require.NoError(t, err)
	require.NotNil(t, vpc)

	require.Equal(t, vpcID, vpc.ID)
	require.NotEmpty(t, vpc.Name)
}
