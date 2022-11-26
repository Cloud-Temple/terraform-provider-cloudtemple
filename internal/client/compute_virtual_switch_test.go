package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_VirtualSwitchList(t *testing.T) {
	ctx := context.Background()
	virtualSwitchs, err := client.Compute().VirtualSwitch().List(ctx, "", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualSwitchs), 1)

	var found bool
	for _, vs := range virtualSwitchs {
		if vs.ID == "6e7b457c-bdb1-4272-8abf-5fd6e9adb8a4" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualSwitchRead(t *testing.T) {
	ctx := context.Background()
	virtualSwitch, err := client.Compute().VirtualSwitch().Read(ctx, "6e7b457c-bdb1-4272-8abf-5fd6e9adb8a4")
	require.NoError(t, err)

	expected := &VirtualSwitch{
		ID:    "6e7b457c-bdb1-4272-8abf-5fd6e9adb8a4",
		Name:  "dvs002-ucs01_FLO-DC-EQX6",
		Moref: "dvs-1044",
	}
	require.Equal(t, expected, virtualSwitch)
}
