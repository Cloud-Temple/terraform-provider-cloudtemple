package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	FeatureId   = "IAM_FEATURE_ID"
	FeatureName = "IAM_FEATURE_NAME"
)

func TestAccDataSourceFeatures(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceFeatures,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des features n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_iam_features.foo",
						"features.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse features count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected features list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_features.foo", "features.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_iam_features.foo", "features.0.name"),
				),
			},
		},
	})
}

const testAccDataSourceFeatures = `
data "cloudtemple_iam_features" "foo" {}
`
