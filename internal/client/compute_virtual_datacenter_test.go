package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	VirtualDatacenterId   = "TEST_COMPUTE_VIRTUAL_DATACENTER_ID"
	VirtualDatacenterName = "TEST_COMPUTE_VIRTUAL_DATACENTER_NAME"
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

	expected := &VirtualDatacenter{
		ID:               os.Getenv(VirtualDatacenterId),
		Name:             os.Getenv(VirtualDatacenterName),
		MachineManagerID: os.Getenv(MachineManagerId),
		TenantID:         os.Getenv(TenantId),
	}
	require.Equal(t, expected, virtualDatacenter)
}
