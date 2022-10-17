package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIAM_Users(t *testing.T) {
	t.Parallel()

	companyID := os.Getenv(testCompanyIDEnvName)
	users, err := client.IAM().User().List(context.Background(), companyID)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(users), 1)

	var found bool
	for _, user := range users {
		if user.Email == "remi.lapeyre@lenstra.fr" {
			found = true
			expected := &User{
				ID:            "37105598-4889-43da-82ea-cf60f2a36aee",
				InternalID:    "7b8ba092-52e3-4c21-a2f5-adca40a80d34",
				Name:          "Rémi Lapeyre",
				Type:          "LocalAccount",
				Source:        nil,
				SourceID:      "",
				EmailVerified: true, Email: "remi.lapeyre@lenstra.fr",
			}
			require.Equal(t, expected, user)
			break
		}
	}
	require.True(t, found)
}

func TestIAM_User(t *testing.T) {
	t.Parallel()

	user, err := client.IAM().User().Read(context.Background(), "37105598-4889-43da-82ea-cf60f2a36aee")
	require.NoError(t, err)

	expected := &User{
		ID:            "37105598-4889-43da-82ea-cf60f2a36aee",
		InternalID:    "7b8ba092-52e3-4c21-a2f5-adca40a80d34",
		Name:          "Rémi Lapeyre",
		Type:          "LocalAccount",
		Source:        nil,
		SourceID:      "",
		EmailVerified: true, Email: "remi.lapeyre@lenstra.fr",
	}
	require.Equal(t, expected, user)
}
