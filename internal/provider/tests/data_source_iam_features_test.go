package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	FeatureId     = "IAM_FEATURE_ID"
	FeatureName   = "IAM_FEATURE_NAME"
	SubFeatureQty = "IAM_SUBFEATURE_QTY"
)

func TestAccDataSourceFeatures(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceFeatures,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_features.foo", "features.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.cloudtemple_iam_features.foo", "features.*", map[string]string{
						"id":   os.Getenv(FeatureId),
						"name": os.Getenv(FeatureName),
					},
					),
				),
			},
		},
	})
}

const testAccDataSourceFeatures = `
data "cloudtemple_iam_features" "foo" {}
`
