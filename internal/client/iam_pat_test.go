package client

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	PatName = "TEST_IAM_PAT_NAME"
)

func TestIAM_PATList(t *testing.T) {
	ctx := context.Background()
	tokens, err := client.IAM().PAT().List(ctx, testUserID(t), testTenantID(t))
	require.NoError(t, err)

	found := false
	for _, token := range tokens {
		if token.Name == os.Getenv(PatName) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestIAM_PATRead(t *testing.T) {
	ctx := context.Background()
	tokens, err := client.IAM().PAT().List(ctx, testUserID(t), testTenantID(t))
	require.NoError(t, err)

	var id string
	for _, token := range tokens {
		if token.Name == os.Getenv(PatName) {
			id = token.ID
			break
		}
	}
	if id == "" {
		t.Fatalf(`failed to find token named "Terraform"`)
	}

	token, err := client.IAM().PAT().Read(ctx, tokens[0].ID)
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
