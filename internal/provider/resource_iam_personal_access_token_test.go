package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourcePersonalAccessToken(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePersonalAccessToken,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_iam_personal_access_token.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttrSet("cloudtemple_iam_personal_access_token.foo", "client_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_iam_personal_access_token.foo", "secret_id"),
				),
			},
			// {
			// 	PreConfig: func() {
			// 		c, userID, tenantID := getTestClient(t)
			// 		tokens, err := c.IAM().PAT().List(context.Background(), userID, tenantID)
			// 		if err != nil {
			// 			t.Fatalf("fail to list tokens: %s", err)
			// 		}

			// 		var tok *client.Token
			// 		for _, token := range tokens {
			// 			if token.Name == "test-terraform" {
			// 				tok = token
			// 				break
			// 			}
			// 		}
			// 		if tok == nil {
			// 			t.Fatalf(`failed to find "test-terraform" terraform`)
			// 		}

			// 		err = c.IAM().PAT().Delete(context.Background(), tok.ID)
			// 		if err != nil {
			// 			t.Fatalf("failed to delete token: %s", err)
			// 		}
			// 	},
			// 	Config: testAccResourcePersonalAccessToken,
			// 	Check:  resource.ComposeTestCheckFunc(),
			// },
		},
	})
}

const testAccResourcePersonalAccessToken = `
resource "cloudtemple_iam_personal_access_token" "foo" {
  name            = "test-terraform"
  expiration_date = "2023-01-02T15:04:05Z"
  
  roles = [
	"c83a22e9-70bb-485e-a463-78a99484e5bb"
  ]
}
`
