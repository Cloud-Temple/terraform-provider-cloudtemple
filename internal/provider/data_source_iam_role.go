package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRole() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceRoleRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},

			// Out
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	roleID := d.Get("id").(string)

	role, err := client.IAM().Role().Read(ctx, roleID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(roleID)

	sw := newStateWriter(d)
	sw.set("name", role.Name)
	return sw.diags
}
