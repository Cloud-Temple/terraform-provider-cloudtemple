package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceContentLibrary() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceContentLibraryReadRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
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
	}
}

func dataSourceContentLibraryReadRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	id := d.Get("id").(string)

	cl, err := client.Compute().ContentLibrary().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, id)
	sw.set("name", cl.Name)
	sw.set("machine_manager_id", cl.MachineManagerID)
	sw.set("type", cl.Type)
	sw.set("datastore", []interface{}{
		map[string]interface{}{
			"id":   cl.Datastore.Name,
			"name": cl.Datastore.Name,
		},
	})

	return sw.diags
}
