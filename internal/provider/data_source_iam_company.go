package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCompany() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceCompanyRead,

		Schema: map[string]*schema.Schema{
			// Out
			"name": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceCompanyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	company, err := client.IAM().Company().Read(ctx, "77a7d0a7-768d-4688-8c32-5fc539c5a859")
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(company.ID)

	sw := newStateWriter(d)
	sw.set("name", company.Name)

	return sw.diags
}
