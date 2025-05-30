package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	IamUserEmail      = "IAM_USER_EMAIL"
	IamUserId         = "IAM_USER_ID"
	IamUserInternalId = "IAM_USER_INTERNAL_ID"
	IamUserName       = "IAM_USER_NAME"
	IamUserType       = "IAM_USER_TYPE"
)

func TestIAM_Users(t *testing.T) {
	companyID := os.Getenv(testCompanyIDEnvName)
	users, err := client.IAM().User().List(context.Background(), &clientpkg.UserFilter{
		CompanyID: companyID,
	})
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
