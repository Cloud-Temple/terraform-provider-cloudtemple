package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRoles() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceRolesRead,

		Schema: map[string]*schema.Schema{
			// Out
			"roles": {
				Description: "",
				Type:        schema.TypeList,
				Computed:    true,

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

func dataSourceRolesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	roles, err := client.IAM().Role().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("roles")
	sw := newStateWriter(d)

	lRoles := []interface{}{}
	for _, r := range roles {
		lRoles = append(lRoles, map[string]interface{}{
			"id":   r.ID,
			"name": r.Name,
		})
	}

	sw.set("roles", lRoles)

	return sw.diags
}
