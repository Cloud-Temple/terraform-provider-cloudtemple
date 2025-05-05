package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	OperatingSystemMoref = "COMPUTE_OPERATING_SYSTEM_MOREF"
)

func TestCompute_GuestOperatingSystemList(t *testing.T) {
	ctx := context.Background()
	folders, err := client.Compute().GuestOperatingSystem().List(ctx, &clientpkg.GuestOperatingSystemFilter{})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(folders), 1)

	var found bool
	for _, f := range folders {
		if f.Moref == os.Getenv(OperatingSystemMoref) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_GuestOperatingSystemRead(t *testing.T) {
	ctx := context.Background()
	folder, err := client.Compute().GuestOperatingSystem().Read(ctx, os.Getenv(OperatingSystemMoref), &clientpkg.GuestOperatingSystemFilter{})
	require.NoError(t, err)

	expected := &clientpkg.GuestOperatingSystem{
		Moref: os.Getenv(OperatingSystemMoref),
	}
	require.Equal(t, expected, folder)
}
