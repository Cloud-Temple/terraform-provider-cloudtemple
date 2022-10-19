package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceFeatures(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceFeatures,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_iam_features.foo", "features.#", "11"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_features.foo", "features.0.id", "5d74006a-ad00-42e0-92b4-5b8d41108641"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_features.foo", "features.0.name", "activity"),
					resource.TestCheckResourceAttr("data.cloudtemple_iam_features.foo", "features.0.subfeatures.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceFeatures = `
data "cloudtemple_iam_features" "foo" {}
`
