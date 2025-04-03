package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	VirtualDatacenterId   = "COMPUTE_VIRTUAL_DATACENTER_ID"
	VirtualDatacenterName = "COMPUTE_VIRTUAL_DATACENTER_NAME"
)

func TestCompute_VirtualDatacenterList(t *testing.T) {
	ctx := context.Background()
	virtualDatacenters, err := client.Compute().VirtualDatacenter().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualDatacenters), 1)

	var found bool
	for _, vd := range virtualDatacenters {
		if vd.ID == os.Getenv(VirtualDatacenterId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualDatacenterRead(t *testing.T) {
	ctx := context.Background()
	virtualDatacenter, err := client.Compute().VirtualDatacenter().Read(ctx, os.Getenv(VirtualDatacenterId))
	require.NoError(t, err)

	expected := &clientpkg.VirtualDatacenter{
		ID:               os.Getenv(VirtualDatacenterId),
		Name:             os.Getenv(VirtualDatacenterName),
		MachineManagerID: os.Getenv(MachineManagerId),
		TenantID:         os.Getenv(TenantId),
	}
	require.Equal(t, expected, virtualDatacenter)
}
