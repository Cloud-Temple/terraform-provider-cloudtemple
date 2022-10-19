package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_VCenterWorkerList(t *testing.T) {
	ctx := context.Background()
	vcenters, err := client.Compute().Worker().List(ctx, "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(vcenters), 1)

	var found bool
	for _, h := range vcenters {
		if h.ID == "9dba240e-a605-4103-bac7-5336d3ffd124" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VCenterWorkerRead(t *testing.T) {
	ctx := context.Background()
	vcenter, err := client.Compute().Worker().Read(ctx, "9dba240e-a605-4103-bac7-5336d3ffd124")
	require.NoError(t, err)

	expected := &Worker{
		ID:                    "9dba240e-a605-4103-bac7-5336d3ffd124",
		Name:                  "vc-vstack-080-bob",
		FullName:              "VMware vCenter Server 6.7.0 build-19832280",
		Vendor:                "VMware, Inc.",
		Version:               "6.7.0",
		Build:                 19832280,
		LocaleVersion:         "INTL",
		LocaleBuild:           0,
		OsType:                "linux-x64",
		ProductLineID:         "vpx",
		ApiType:               "VirtualCenter",
		ApiVersion:            "6.7.3",
		InstanceUuid:          "e919d66b-cac9-4ea5-839e-74668f49958c",
		LicenseProductName:    "VMware VirtualCenter Server",
		LicenseProductVersion: 6,
		TenantID:              "e225dbf8-e7c5-4664-a595-08edf3526080",
		TenantName:            "BOB",
	}
	require.Equal(t, expected, vcenter)
}
