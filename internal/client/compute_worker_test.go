package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	TenantId               = "TEST_TENANT_ID"
	TenantName             = "TEST_TENANT_NAME"
	ComputeVCenterId       = "TEST_COMPUTE_VCENTER_ID"
	ComputeVCenterName     = "TEST_COMPUTE_VCENTER_NAME"
	ComputeVCenterFullName = "TEST_COMPUTE_VCENTER_FULLNAME"
	ComputeVCenterVendor   = "TEST_COMPUTE_VCENTER_VENDOR"
	ComputeVCenterVersion  = "TEST_COMPUTE_VCENTER_VERSION"
)

func TestCompute_VCenterWorkerList(t *testing.T) {
	ctx := context.Background()
	vcenters, err := client.Compute().Worker().List(ctx, "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(vcenters), 1)

	var found bool
	for _, h := range vcenters {
		if h.ID == os.Getenv(ComputeVCenterId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VCenterWorkerRead(t *testing.T) {
	ctx := context.Background()
	vcenter, err := client.Compute().Worker().Read(ctx, os.Getenv(ComputeVCenterId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(ComputeVCenterId), vcenter.ID)
	require.Equal(t, os.Getenv(ComputeVCenterName), vcenter.Name)
	require.Equal(t, os.Getenv(ComputeVCenterFullName), vcenter.FullName)
	require.Equal(t, os.Getenv(ComputeVCenterVendor), vcenter.Vendor)
	require.Equal(t, os.Getenv(ComputeVCenterVersion), vcenter.Version)
	require.Equal(t, os.Getenv(TenantId), vcenter.TenantID)
}
