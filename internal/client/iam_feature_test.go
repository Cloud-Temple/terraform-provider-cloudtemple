package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIAM_Features(t *testing.T) {
	t.Parallel()

	features, err := client.IAM().Feature().List(context.Background())
	require.NoError(t, err)

	var rtms *Feature
	for _, feature := range features {
		if feature.Name == "rtms" {
			rtms = feature
			break
		}
	}
	require.NotNil(t, rtms, "rtms has not been found")
	require.Equal(t, "rtms", rtms.Name)
	require.Equal(t, "f39df526-66c5-465b-a52f-29180e241e09", rtms.ID)
	require.Len(t, rtms.SubFeatures, 2)
}

func TestIAM_FeatureAssignments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenants, err := client.IAM().Tenant().List(ctx)
	require.NoError(t, err)
	require.Len(t, tenants, 1)

	tenant := tenants[0]
	fa, err := client.IAM().Feature().ListAssignments(ctx, tenant.ID)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(fa), 1)
	require.NotEqual(t, "", fa[0].FeatureID)
	require.NotEqual(t, "", fa[0].TenantID)
}
