package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceContentLibraries() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceContentLibrariesRead,

		Schema: map[string]*schema.Schema{
			"content_libraries": {
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
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"datastore": {
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

func dataSourceContentLibrariesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	contentLibraries, err := client.Compute().ContentLibrary().List(ctx, "", "", "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(contentLibraries))
	for i, cl := range contentLibraries {
		res[i] = map[string]interface{}{
			"id":                 cl.ID,
			"name":               cl.Name,
			"machine_manager_id": cl.MachineManagerID,
			"type":               cl.Type,
			"datastore": []interface{}{
				map[string]interface{}{
					"id":   cl.Datastore.ID,
					"name": cl.Datastore.Name,
				},
			},
		}
	}

	sw := newStateWriter(d, "content-libraries")
	sw.set("content_libraries", res)

	return sw.diags
}
