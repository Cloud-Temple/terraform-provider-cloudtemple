package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_DatastoreList(t *testing.T) {
	ctx := context.Background()
	datastores, err := client.Compute().Datastore().List(ctx, "", "", "", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(datastores), 1)

	var found bool
	for _, dc := range datastores {
		if dc.ID == "d439d467-943a-49f5-a022-c0c25b737022" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_DatastoreRead(t *testing.T) {
	ctx := context.Background()
	datastore, err := client.Compute().Datastore().Read(ctx, "d439d467-943a-49f5-a022-c0c25b737022")
	require.NoError(t, err)

	expected := &Datastore{
		ID:                    "d439d467-943a-49f5-a022-c0c25b737022",
		Name:                  "ds001-bob-svc1-data4-eqx6",
		Moref:                 "datastore-1056",
		MaxCapacity:           536602476544,
		FreeCapacity:          131042639872,
		Accessible:            1,
		MaintenanceStatus:     "normal",
		UniqueId:              "601d9902-28268458-ac1a-0025b553004c",
		MachineManagerId:      "9dba240e-a605-4103-bac7-5336d3ffd124",
		Type:                  "VMFS",
		VirtualMachinesNumber: 5,
		HostsNumber:           1,
		HostsNames:            nil,
		AssociatedFolder:      "",
	}
	require.Equal(t, expected, datastore)
}
