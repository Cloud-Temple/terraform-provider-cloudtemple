package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_DatastoreList(t *testing.T) {
	ctx := context.Background()
	datastores, err := client.Compute().Datastore().List(ctx, &DatastoreFilter{
		DatacenterId: "7b56f202-83e3-4112-9771-8fb001fbac3e",
	})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(datastores), 1)

	var found bool
	for _, dc := range datastores {
		if dc.ID == "88fb9089-cf33-47f0-938a-fe792f4a9039" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_DatastoreRead(t *testing.T) {
	ctx := context.Background()
	datastore, err := client.Compute().Datastore().Read(ctx, "88fb9089-cf33-47f0-938a-fe792f4a9039")
	require.NoError(t, err)

	// Skip checking changes on metrics
	datastore.FreeCapacity = 0
	datastore.VirtualMachinesNumber = 0

	expected := &Datastore{
		ID:                "d439d467-943a-49f5-a022-c0c25b737022",
		Name:              "ds001-bob-svc1-data4-eqx6",
		Moref:             "datastore-1056",
		MaxCapacity:       536602476544,
		Accessible:        1,
		MaintenanceStatus: "normal",
		UniqueId:          "601d9902-28268458-ac1a-0025b553004c",
		MachineManagerId:  "9dba240e-a605-4103-bac7-5336d3ffd124",
		Type:              "VMFS",
		HostsNumber:       1,
		HostsNames:        nil,
		AssociatedFolder:  "",
	}
	require.Equal(t, expected, datastore)
}
