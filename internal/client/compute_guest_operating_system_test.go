package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	OperationSystemMoref    = "COMPUTE_OPERATION_SYSTEM_MOREF"
	OperationSystemFamily   = "COMPUTE_OPERATION_SYSTEM_FAMILY"
	OperationSystemFullName = "COMPUTE_OPERATION_SYSTEM_FULLNAME"
)

func TestCompute_GuestOperatingSystemList(t *testing.T) {
	ctx := context.Background()
	folders, err := client.Compute().GuestOperatingSystem().List(ctx, os.Getenv(MachineManagerId), "", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(folders), 1)

	var found bool
	for _, f := range folders {
		if f.Moref == os.Getenv(OperationSystemMoref) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_GuestOperatingSystemRead(t *testing.T) {
	ctx := context.Background()
	folder, err := client.Compute().GuestOperatingSystem().Read(ctx, os.Getenv(MachineManagerId), os.Getenv(OperationSystemMoref))
	require.NoError(t, err)

	expected := &GuestOperatingSystem{
		Moref:    os.Getenv(OperationSystemMoref),
		Family:   os.Getenv(OperationSystemFamily),
		FullName: os.Getenv(OperationSystemFullName),
	}
	require.Equal(t, expected, folder)
}
