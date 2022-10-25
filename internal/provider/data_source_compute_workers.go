package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWorkers() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceWorkersRead,

		Schema: map[string]*schema.Schema{
			"workers": {
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
						"full_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vendor": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"build": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"locale_version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"locale_build": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"os_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"product_line_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"api_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"api_version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"instance_uuid": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"license_product_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"license_product_version": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"tenant_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"tenant_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceWorkersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	workers, err := client.Compute().Worker().List(ctx, "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(workers))
	for i, w := range workers {
		res[i] = map[string]interface{}{
			"id":                      w.ID,
			"name":                    w.Name,
			"full_name":               w.FullName,
			"vendor":                  w.Vendor,
			"version":                 w.Version,
			"build":                   w.Build,
			"locale_version":          w.LocaleVersion,
			"locale_build":            w.LocaleBuild,
			"os_type":                 w.OsType,
			"product_line_id":         w.ProductLineID,
			"api_type":                w.ApiType,
			"api_version":             w.ApiVersion,
			"instance_uuid":           w.InstanceUuid,
			"license_product_name":    w.LicenseProductName,
			"license_product_version": w.LicenseProductVersion,
			"tenant_id":               w.TenantID,
			"tenant_name":             w.TenantName,
		}
	}

	sw := newStateWriter(d, "workers")
	sw.set("workers", res)

	return sw.diags
}
