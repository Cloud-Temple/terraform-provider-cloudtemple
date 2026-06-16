package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenIaasCreateRequestsOmitEmptyCloudInit pins the fix for the empty
// cloudInit bug: an unset CloudInit must be OMITTED from the request body. The
// API rejects "cloudInit":{} with "When cloudInit is provided, cloudConfig is
// required", so a value-typed field (which omitempty cannot drop) breaks every
// deploy without cloud-init. This test goes RED if the field reverts to a value
// struct.
func TestOpenIaasCreateRequestsOmitEmptyCloudInit(t *testing.T) {
	m, err := json.Marshal(&MarketplaceOpenIaasDeployementRequest{ID: "id", Name: "n", StorageRepositoryID: "sr"})
	require.NoError(t, err)
	require.NotContains(t, string(m), "cloudInit", "marketplace deploy must omit an unset cloudInit (not send {})")

	c, err := json.Marshal(&CreateOpenIaasVirtualMachineRequest{Name: "n", TemplateID: "t"})
	require.NoError(t, err)
	require.NotContains(t, string(c), "cloudInit", "template create must omit an unset cloudInit (not send {})")
}

// TestOpenIaasCreateRequestsIncludeCloudInitWhenSet documents the positive path:
// a configured cloud-init is serialized with its cloudConfig.
func TestOpenIaasCreateRequestsIncludeCloudInitWhenSet(t *testing.T) {
	m, err := json.Marshal(&MarketplaceOpenIaasDeployementRequest{
		ID:        "id",
		CloudInit: &CloudInit{CloudConfig: "#cloud-config\npackages: [htop]"},
	})
	require.NoError(t, err)
	require.Contains(t, string(m), `"cloudInit"`)
	require.Contains(t, string(m), `"cloudConfig"`)

	c, err := json.Marshal(&CreateOpenIaasVirtualMachineRequest{
		Name:      "n",
		CloudInit: &CloudInit{CloudConfig: "#cloud-config", NetworkConfig: "version: 2"},
	})
	require.NoError(t, err)
	require.Contains(t, string(c), `"cloudInit"`)
	require.Contains(t, string(c), `"networkConfig"`)
}
