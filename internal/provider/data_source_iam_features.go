package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFeatures() *schema.Resource {
	return &schema.Resource{
		Description: "",

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
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceFeaturesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	features, err := client.IAM().Feature().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("features")

	var res []interface{}
	for _, feat := range features {
		var sres []interface{}
		for _, subfeat := range feat.SubFeatures {
			sres = append(sres, map[string]interface{}{
				"id":   subfeat.ID,
				"name": subfeat.Name,
			})
		}

		res = append(res, map[string]interface{}{
			"id":          feat.ID,
			"name":        feat.Name,
			"subfeatures": sres,
		})
	}

	sw := newStateWriter(d)
	sw.set("features", res)
	return sw.diags
}
