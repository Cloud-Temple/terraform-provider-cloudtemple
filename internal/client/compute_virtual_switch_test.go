package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	ComputeVirtualSwitchId    = "TEST_COMPUTE_VIRTUAL_SWITCH_ID"
	ComputeVirtualSwitchName  = "TEST_COMPUTE_VIRTUAL_SWITCH_NAME"
	ComputeVirtualSwitchMoref = "TEST_COMPUTE_VIRTUAL_SWITCH_MOREF"
)

func TestCompute_VirtualSwitchList(t *testing.T) {
	ctx := context.Background()
	virtualSwitchs, err := client.Compute().VirtualSwitch().List(ctx, "", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualSwitchs), 1)

	var found bool
	for _, vs := range virtualSwitchs {
		if vs.ID == os.Getenv(ComputeVirtualSwitchId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualSwitchRead(t *testing.T) {
	ctx := context.Background()
	virtualSwitch, err := client.Compute().VirtualSwitch().Read(ctx, os.Getenv(ComputeVirtualSwitchId))
	require.NoError(t, err)

	expected := &VirtualSwitch{
		ID:    os.Getenv(ComputeVirtualSwitchId),
		Name:  os.Getenv(ComputeVirtualSwitchName),
		Moref: os.Getenv(ComputeVirtualSwitchMoref),
	}
	require.Equal(t, expected, virtualSwitch)
}
