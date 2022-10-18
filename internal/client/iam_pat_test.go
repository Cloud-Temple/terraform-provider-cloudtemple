package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIAM_PATList(t *testing.T) {
	ctx := context.Background()
	tokens, err := client.IAM().PAT().List(ctx, testUserID(t), testTenantID(t))
	require.NoError(t, err)

	require.Len(t, tokens, 1)
	require.Equal(t, "Terraform", tokens[0].Name)
}

func TestIAM_PATRead(t *testing.T) {
	ctx := context.Background()
	tokens, err := client.IAM().PAT().List(ctx, testUserID(t), testTenantID(t))
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	token, err := client.IAM().PAT().Read(ctx, tokens[0].ID)
	require.NoError(t, err)
	require.Equal(t, "Terraform", token.Name)
	require.Equal(t, "", token.Secret)
}

func TestIAM_PATCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	roles := []string{testRole(t).ID}
	expirationDate := int(time.Now().UnixMilli() + 24*60*60*1000)
	token, err := client.IAM().PAT().Create(ctx, "client-test", roles, expirationDate)
	require.NoError(t, err)
	require.NotEmpty(t, token.Secret)

	err = client.IAM().PAT().Delete(ctx, token.ID)
	require.NoError(t, err)
}
