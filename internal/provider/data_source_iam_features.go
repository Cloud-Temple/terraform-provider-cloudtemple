package provider

import (
	"context"

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
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"subfeatures": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"name": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"subfeatures": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"id": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"name": {
													Type:     schema.TypeString,
													Computed: true,
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

	// Mapper les données en utilisant la fonction helper
	tfFeatures := make([]map[string]interface{}, len(features))
	for i, feature := range features {
		tfFeatures[i] = helpers.FlattenFeature(feature)
	}

	// Définir les données dans le state
	if err := d.Set("features", tfFeatures); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
