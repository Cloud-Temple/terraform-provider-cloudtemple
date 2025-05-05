package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

var virtualControllerId = ""

func TestCompute_VirtualControllerList(t *testing.T) {
	ctx := context.Background()
	virtualControllers, err := client.Compute().VirtualController().List(ctx, &clientpkg.VirtualControllerFilter{
		VirtualMachineId: os.Getenv(VirtualMachineId),
	})
	require.NoError(t, err)

	virtualControllerId = virtualControllers[0].ID

	require.GreaterOrEqual(t, len(virtualControllers), 1)
}

func TestCompute_VirtualControllerRead(t *testing.T) {
	ctx := context.Background()
	virtualController, err := client.Compute().VirtualController().Read(ctx, virtualControllerId)
	require.NoError(t, err)

	require.Equal(t, virtualControllerId, virtualController.ID)
	require.NotEmpty(t, virtualController.Label)
	require.NotEmpty(t, virtualController.Type)
	require.NotEmpty(t, virtualController.VirtualMachineId)
}
