package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	IamUserEmail      = "TEST_IAM_USER_EMAIL"
	IamUserId         = "TEST_IAM_USER_ID"
	IamUserInternalId = "TEST_IAM_USER_INTERNAL_ID"
	IamUserName       = "TEST_IAM_USER_NAME"
	IamUserType       = "TEST_IAM_USER_TYPE"
)

func TestIAM_Users(t *testing.T) {
	companyID := os.Getenv(testCompanyIDEnvName)
	users, err := client.IAM().User().List(context.Background(), companyID)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(users), 1)

	var found bool
	for _, user := range users {

		if user.Email == os.Getenv(IamUserEmail) {
			found = true
			require.Equal(t, os.Getenv(IamUserId), user.ID)
			require.Equal(t, os.Getenv(IamUserName), user.Name)
			require.Equal(t, os.Getenv(IamUserType), user.Type)
			require.Equal(t, os.Getenv(IamUserInternalId), user.InternalID)
			break
		}
	}
	require.True(t, found)
}

func TestIAM_User(t *testing.T) {
	user, err := client.IAM().User().Read(context.Background(), os.Getenv(IamUserId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(IamUserId), user.ID)
	require.Equal(t, os.Getenv(IamUserName), user.Name)
	require.Equal(t, os.Getenv(IamUserType), user.Type)
	require.Equal(t, os.Getenv(IamUserInternalId), user.InternalID)
}
