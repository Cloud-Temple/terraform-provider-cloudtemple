package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildOpenIaasCloudInit pins the cloud-init build logic that feeds the
// OpenIaaS create payload (#345): omit when unconfigured (so "cloudInit" is not
// sent as an empty object the API rejects), fail closed when network_config is
// set without cloud_config, and populate when cloud_config is set.
func TestBuildOpenIaasCloudInit(t *testing.T) {
	t.Run("absent (nil map) → nil, no error", func(t *testing.T) {
		ci, err := buildOpenIaasCloudInit(nil)
		require.NoError(t, err)
		require.Nil(t, ci)
	})

	t.Run("empty map → nil", func(t *testing.T) {
		ci, err := buildOpenIaasCloudInit(map[string]interface{}{})
		require.NoError(t, err)
		require.Nil(t, ci)
	})

	t.Run("all-empty values → nil", func(t *testing.T) {
		ci, err := buildOpenIaasCloudInit(map[string]interface{}{"cloud_config": "", "network_config": ""})
		require.NoError(t, err)
		require.Nil(t, ci, "an all-empty cloud_init must be omitted, not sent as {}")
	})

	t.Run("network_config without cloud_config → error, fail closed", func(t *testing.T) {
		ci, err := buildOpenIaasCloudInit(map[string]interface{}{"network_config": "version: 2"})
		require.Error(t, err, "cloud_config is required when cloud_init is set; must fail before the API 400")
		require.Nil(t, ci)
	})

	t.Run("cloud_config only → populated, no network_config", func(t *testing.T) {
		ci, err := buildOpenIaasCloudInit(map[string]interface{}{"cloud_config": "#cloud-config"})
		require.NoError(t, err)
		require.NotNil(t, ci)
		require.Equal(t, "#cloud-config", ci.CloudConfig)
		require.Empty(t, ci.NetworkConfig)
	})

	t.Run("cloud_config + network_config → populated both", func(t *testing.T) {
		ci, err := buildOpenIaasCloudInit(map[string]interface{}{"cloud_config": "#cloud-config", "network_config": "version: 2"})
		require.NoError(t, err)
		require.NotNil(t, ci)
		require.Equal(t, "#cloud-config", ci.CloudConfig)
		require.Equal(t, "version: 2", ci.NetworkConfig)
	})
}
