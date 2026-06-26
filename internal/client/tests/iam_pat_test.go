package client

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	PatName = "IAM_PAT_NAME"
)

func TestIAM_PATList(t *testing.T) {
	ctx := context.Background()
	lt, err := client.Token(ctx)
	require.NoError(t, err)
	tokens, err := client.IAM().PAT().List(ctx)
	require.NoError(t, err)

	found := false
	for _, token := range tokens {
		// List(ctx) is unscoped (see #226); match the principal's own token.
		if token.Name == os.Getenv(PatName) && token.UserId == lt.UserID() && token.TenantId == lt.TenantID() {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestIAM_PATRead(t *testing.T) {
	ctx := context.Background()
	lt, err := client.Token(ctx)
	require.NoError(t, err)
	tokens, err := client.IAM().PAT().List(ctx)
	require.NoError(t, err)

	var id string
	for _, token := range tokens {
		// List(ctx) is unscoped (see #226); match the principal's own token.
		if token.Name == os.Getenv(PatName) && token.UserId == lt.UserID() && token.TenantId == lt.TenantID() {
			id = token.ID
			break
		}
	}
	if id == "" {
		t.Fatalf(`failed to find token named "Terraform"`)
	}

	token, err := client.IAM().PAT().Read(ctx, id)
	require.NoError(t, err)
	require.Equal(t, os.Getenv(PatName), token.Name)
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
