package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWorker() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceWorkerRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
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
	}
}

func dataSourceWorkerRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	worker, err := client.Compute().Worker().Read(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, worker.ID)
	sw.set("id", worker.ID)
	sw.set("name", worker.Name)
	sw.set("full_name", worker.FullName)
	sw.set("vendor", worker.Vendor)
	sw.set("version", worker.Version)
	sw.set("build", worker.Build)
	sw.set("locale_version", worker.LocaleVersion)
	sw.set("locale_build", worker.LocaleBuild)
	sw.set("os_type", worker.OsType)
	sw.set("product_line_id", worker.ProductLineID)
	sw.set("api_type", worker.ApiType)
	sw.set("api_version", worker.ApiVersion)
	sw.set("instance_uuid", worker.InstanceUuid)
	sw.set("license_product_name", worker.LicenseProductName)
	sw.set("license_product_version", worker.LicenseProductVersion)
	sw.set("tenant_id", worker.TenantID)
	sw.set("tenant_name", worker.TenantName)

	return sw.diags
}
