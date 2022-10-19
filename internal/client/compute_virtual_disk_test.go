package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_VirtualDiskList(t *testing.T) {
	ctx := context.Background()
	virtualDisks, err := client.Compute().VirtualDisk().List(ctx, "de2b8b80-8b90-414a-bc33-e12f61a4c05c")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualDisks), 1)

	var found bool
	for _, vd := range virtualDisks {
		if vd.ID == "d370b8cd-83eb-4315-a5d9-42157e2e4bb4" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualDiskRead(t *testing.T) {
	ctx := context.Background()
	virtualDatacenter, err := client.Compute().VirtualDisk().Read(ctx, "d370b8cd-83eb-4315-a5d9-42157e2e4bb4")
	require.NoError(t, err)

	expected := &VirtualDisk{
		ID:                  "d370b8cd-83eb-4315-a5d9-42157e2e4bb4",
		VirtualMachineId:    "de2b8b80-8b90-414a-bc33-e12f61a4c05c",
		MachineManagerId:    "9dba240e-a605-4103-bac7-5336d3ffd124",
		Name:                "Hard disk 1",
		Capacity:            17179869184,
		DiskUnitNumber:      0,
		ControllerBusNumber: 0,
		DatastoreId:         "d439d467-943a-49f5-a022-c0c25b737022",
		DatastoreName:       "ds001-bob-svc1-data4-eqx6",
		InstantAccess:       false,
		NativeId:            "6000C296-f0b9-c149-21f6-a1877fc8bae8",
		DiskPath:            "[ds001-bob-svc1-data4-eqx6] virtual_machine_67_bob-clone/virtual_machine_67_bob-clone_2.vmdk",
		ProvisioningType:    "staticDiffered",
		DiskMode:            "persistent",
		Editable:            true,
	}
	require.Equal(t, expected, virtualDatacenter)
}
