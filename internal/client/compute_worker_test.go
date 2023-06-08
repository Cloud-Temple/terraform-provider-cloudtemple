package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	TenantId        = "TENANT_ID"
	TenantName      = "TENANT_NAME"
	VCenterId       = "COMPUTE_VCENTER_ID"
	VCenterName     = "COMPUTE_VCENTER_NAME"
	VCenterFullName = "COMPUTE_VCENTER_FULLNAME"
	VCenterVendor   = "COMPUTE_VCENTER_VENDOR"
	VCenterVersion  = "COMPUTE_VCENTER_VERSION"
)

func TestCompute_VCenterWorkerList(t *testing.T) {
	ctx := context.Background()
	vcenters, err := client.Compute().Worker().List(ctx, "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(vcenters), 1)

	var found bool
	for _, h := range vcenters {
		if h.ID == os.Getenv(VCenterId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VCenterWorkerRead(t *testing.T) {
	ctx := context.Background()
	vcenter, err := client.Compute().Worker().Read(ctx, os.Getenv(VCenterId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(VCenterId), vcenter.ID)
	require.Equal(t, os.Getenv(VCenterName), vcenter.Name)
	require.Equal(t, os.Getenv(VCenterFullName), vcenter.FullName)
	require.Equal(t, os.Getenv(VCenterVendor), vcenter.Vendor)
	require.Equal(t, os.Getenv(VCenterVersion), vcenter.Version)
	require.Equal(t, os.Getenv(TenantId), vcenter.TenantID)
}
