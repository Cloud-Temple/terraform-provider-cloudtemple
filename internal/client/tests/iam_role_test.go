package client

import (
	"context"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

func TestIAM_RolesList(t *testing.T) {
	roles, err := client.IAM().Role().List(context.Background())
	require.NoError(t, err)

	var found bool
	for _, r := range roles {
		if r.Name == "iam_read" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestIAM_RolesRead(t *testing.T) {
	roleID := testRole(t).ID
	role, err := client.IAM().Role().Read(context.Background(), roleID)
	require.NoError(t, err)

	expected := &clientpkg.Role{
		ID:   roleID,
		Name: "iam_read",
	}
	require.Equal(t, expected, role)
}
