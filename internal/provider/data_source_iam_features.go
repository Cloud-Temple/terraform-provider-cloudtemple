package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFeatures() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all available features in the platform.",

		ReadContext: dataSourceFeaturesRead,

		Schema: map[string]*schema.Schema{
			// Out
			"features": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of all available features in the platform.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the feature.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the feature.",
						},
						"subfeatures": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of subfeatures belonging to this feature.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The unique identifier of the subfeature.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the subfeature.",
									},
									"subfeatures": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "List of nested subfeatures belonging to this subfeature.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"id": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The unique identifier of the nested subfeature.",
												},
												"name": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The name of the nested subfeature.",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// dataSourceFeaturesRead lit les features et les mappe dans le state Terraform
func dataSourceFeaturesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les features
	features, err := c.IAM().Feature().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("features")

	// Mapper les données en utilisant la fonction helper. La forme imbriquée des
	// sous-features est bornée à la profondeur déclarée par le schéma ci-dessus
	// (cf. helpers.FlattenFeature) : un arbre plus profond est tronqué — la
	// datasource reste lisible — et signalé par un warning, plutôt que de casser
	// la lecture avec "Invalid address to set" (classe #243).
	tfFeatures := make([]map[string]interface{}, 0, len(features))
	for _, feature := range features {
		if feature == nil {
			continue // un élément null renvoyé par l'API ne doit pas faire paniquer le flatten
		}
		tfFeatures = append(tfFeatures, helpers.FlattenFeature(feature))
		if helpers.FeatureExceedsDeclaredDepth(feature) {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Feature sub-tree deeper than the schema can represent; truncated",
				Detail: fmt.Sprintf(
					"Feature %q (id %s) has sub-features nested deeper than the cloudtemple_iam_features schema represents; the deepest levels were truncated from the datasource output. Please report this to the provider maintainers so the schema can be extended.",
					feature.Name, feature.ID),
			})
		}
	}

	// Définir les données dans le state
	if err := d.Set("features", tfFeatures); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
