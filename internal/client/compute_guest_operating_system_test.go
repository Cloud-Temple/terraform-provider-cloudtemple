package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_GuestOperatingSystemList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	folders, err := client.Compute().GuestOperatingSystem().List(ctx, "9dba240e-a605-4103-bac7-5336d3ffd124", "", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(folders), 1)

	var found bool
	for _, f := range folders {
		if f.Moref == "amazonlinux2_64Guest" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_GuestOperatingSystemRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	folder, err := client.Compute().GuestOperatingSystem().Read(ctx, "9dba240e-a605-4103-bac7-5336d3ffd124", "amazonlinux2_64Guest")
	require.NoError(t, err)

	expected := &GuestOperatingSystem{
		Moref:    "amazonlinux2_64Guest",
		Family:   "linuxGuest",
		FullName: "Amazon Linux 2 (64-bit)",
	}
	require.Equal(t, expected, folder)
}
