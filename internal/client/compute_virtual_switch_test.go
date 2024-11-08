package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	VirtualSwitchId    = "COMPUTE_VIRTUAL_SWITCH_ID"
	VirtualSwitchName  = "COMPUTE_VIRTUAL_SWITCH_NAME"
	VirtualSwitchMoref = "COMPUTE_VIRTUAL_SWITCH_MOREF"
)

func TestCompute_VirtualSwitchList(t *testing.T) {
	ctx := context.Background()
	virtualSwitchs, err := client.Compute().VirtualSwitch().List(ctx, &VirtualSwitchFilter{})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualSwitchs), 1)

	var found bool
	for _, vs := range virtualSwitchs {
		if vs.ID == os.Getenv(VirtualSwitchId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualSwitchRead(t *testing.T) {
	ctx := context.Background()
	virtualSwitch, err := client.Compute().VirtualSwitch().Read(ctx, os.Getenv(VirtualSwitchId))
	require.NoError(t, err)

	expected := &VirtualSwitch{
		ID:    os.Getenv(VirtualSwitchId),
		Name:  os.Getenv(VirtualSwitchName),
		Moref: os.Getenv(VirtualSwitchMoref),
	}
	require.Equal(t, expected, virtualSwitch)
}
