package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIAM_AssignmentList(t *testing.T) {
	t.Parallel()

	userId := testUserID(t)
	tenantID := testTenantID(t)
	assignments, err := client.IAM().Assignment().List(context.Background(), userId, tenantID, "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(assignments), 1)
	require.NotEmpty(t, assignments[0].UserID)
	require.NotEmpty(t, assignments[0].TenantID)
	require.NotEmpty(t, assignments[0].RoleID)

	role := testRole(t)
	assignments, err = client.IAM().Assignment().List(context.Background(), userId, tenantID, role.ID)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(assignments), 1)
	require.NotEmpty(t, assignments[0].UserID)
	require.NotEmpty(t, assignments[0].TenantID)
	require.NotEmpty(t, assignments[0].RoleID)
}
