package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	VirtualControllerId    = "COMPUTE_VIRTUAL_CONTROLLER_ID"
	VirtualControllerType  = "COMPUTE_VIRTUAL_CONTROLLER_TYPE"
	VirtualControllerLabel = "COMPUTE_VIRTUAL_CONTROLLER_LABEL"
)

func TestCompute_VirtualControllerList(t *testing.T) {
	ctx := context.Background()
	virtualControllers, err := client.Compute().VirtualController().List(ctx, os.Getenv(VirtualMachineId), "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualControllers), 1)

	var virtualController *clientpkg.VirtualController
	for _, vc := range virtualControllers {
		if vc.ID == os.Getenv(VirtualControllerId) {
			virtualController = vc
			break
		}
	}
	require.NotNil(t, virtualController)

	require.Equal(t, os.Getenv(VirtualControllerId), virtualController.ID)
	require.Equal(t, os.Getenv(VirtualControllerLabel), virtualController.Label)
	require.Equal(t, os.Getenv(VirtualControllerType), virtualController.Type)
	require.Equal(t, os.Getenv(VirtualMachineId), virtualController.VirtualMachineId)
}
