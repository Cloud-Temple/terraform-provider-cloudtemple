package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_VirtualControllerList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	virtualControllers, err := client.Compute().VirtualController().List(ctx, "de2b8b80-8b90-414a-bc33-e12f61a4c05c", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualControllers), 1)

	var virtualController *VirtualController
	for _, vc := range virtualControllers {
		if vc.ID == "50b70cbc-9d77-4e0d-825d-b44bba78197e" {
			virtualController = vc
			break
		}
	}
	require.NotNil(t, virtualController)

	expected := &VirtualController{
		ID:               "50b70cbc-9d77-4e0d-825d-b44bba78197e",
		VirtualMachineId: "de2b8b80-8b90-414a-bc33-e12f61a4c05c",
		HotAddRemove:     true,
		Type:             "SCSI",
		Label:            "SCSI controller 0",
		Summary:          "LSI Logic",
		VirtualDisks:     []string{"d370b8cd-83eb-4315-a5d9-42157e2e4bb4"},
	}
	require.Equal(t, expected, virtualController)
}
