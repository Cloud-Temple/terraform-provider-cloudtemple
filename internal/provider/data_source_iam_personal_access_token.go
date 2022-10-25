package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePersonalAccessToken() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourcePersonalAccessTokenRead,

		Schema: map[string]*schema.Schema{
			// In
			"client_id": {
				Description: "",
				Type:        schema.TypeString,
				Required:    true,
			},

			// Out
			"name": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"roles": {
				Description: "",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"expiration_date": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourcePersonalAccessTokenRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	ID := d.Get("client_id").(string)

	token, err := client.IAM().PAT().Read(ctx, ID)
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, token.ID)

	sw.set("name", token.Name)
	sw.set("roles", token.Roles)
	sw.set("expiration_date", token.ExpirationDate)

	return sw.diags
}
