package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGuestOperatingSystem() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceGuestOperatingSystemRead,

		Schema: map[string]*schema.Schema{
			"moref": {
				Type:     schema.TypeString,
				Required: true,
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"family": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"full_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceGuestOperatingSystemRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	gos, err := client.Compute().GuestOperatingSystem().Read(ctx, d.Get("machine_manager_id").(string), d.Get("moref").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, gos.Moref)
	sw.set("moref", gos.Moref)
	sw.set("family", gos.Family)
	sw.set("full_name", gos.FullName)

	return sw.diags
}
