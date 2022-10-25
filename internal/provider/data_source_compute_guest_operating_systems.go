package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGuestOperatingSystems() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceGuestOperatingSystemsRead,

		Schema: map[string]*schema.Schema{
			"machine_manager_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"guest_operating_systems": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"moref": {
							Type:     schema.TypeString,
							Computed: true,
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
				},
			},
		},
	}
}

func dataSourceGuestOperatingSystemsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	gos, err := client.Compute().GuestOperatingSystem().List(ctx, d.Get("machine_manager_id").(string), "", "", "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(gos))
	for i, g := range gos {
		res[i] = map[string]interface{}{
			"moref":     g.Moref,
			"family":    g.Family,
			"full_name": g.FullName,
		}
	}

	sw := newStateWriter(d, "guest-operating-systems")
	sw.set("guest_operating_systems", res)

	return sw.diags
}
