package client

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testClientIDEnvName  = "SHIVA_TEST_CLIENT_ID"
	testSecretIDEnvName  = "SHIVA_TEST_SECRET_ID"
	testCompanyIDEnvName = "SHIVA_TEST_COMPANY_ID"
)

var client *Client = nil

func TestMain(m *testing.M) {
	envNames := []string{
		testClientIDEnvName,
		testSecretIDEnvName,
		testCompanyIDEnvName,
	}

	var fail bool
	for _, name := range envNames {
		if os.Getenv(name) == "" {
			fmt.Fprintf(os.Stderr, "%s must be set to run the tests\n", name)
			fail = true
		}
	}
	if fail {
		os.Exit(1)
	}

	config := DefaultConfig()
	config.ClientID = os.Getenv(testClientIDEnvName)
	config.SecretID = os.Getenv(testSecretIDEnvName)

	config.errorOnUnexpectedActivity = true

	c, err := NewClient(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	client = c

	os.Exit(m.Run())
}

func testTenantID(t *testing.T) string {
	t.Helper()

	tenants, err := client.IAM().Tenant().List(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(tenants), 1)

	return tenants[0].ID
}

func testUserID(t *testing.T) string {
	t.Helper()

	lt, err := client.Token(context.Background())
	require.NoError(t, err)

	return lt.UserID()
}

func testRole(t *testing.T) *Role {
	t.Helper()

	roles, err := client.IAM().Role().List(context.Background())
	require.NoError(t, err)

	var role *Role
	for _, r := range roles {
		if r.Name == "iam_read" {
			role = r
			break
		}
	}
	require.NotNil(t, role)

	return role
}

func TestAPI_token(t *testing.T) {
	token, err := client.token(context.Background())
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotNil(t, client.savedToken)
}

func TestAPI_tokenCache(t *testing.T) {
	token, err := client.token(context.Background())
	require.NoError(t, err)

	newToken, err := client.token(context.Background())
	require.NoError(t, err)

	require.Equal(t, token.Raw, newToken.Raw)
}
