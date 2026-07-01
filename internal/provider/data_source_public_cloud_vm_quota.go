package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePublicCloudVMQuota() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve the Public Cloud VM Instances quota of the tenant (limits and current usage). The tenant is derived from the JWT scope; this datasource takes no argument.",

		ReadContext: publicCloudVMQuotaRead,

		Schema: map[string]*schema.Schema{
			// Out
			"vcpu_limit": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The maximum number of vCPUs allowed for the tenant.",
			},
			"ram_limit_mb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The maximum amount of RAM allowed for the tenant, in MB.",
			},
			"storage_limit_gb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The maximum amount of storage allowed for the tenant, in GB.",
			},
			"vcpu_used": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of vCPUs currently consumed by the tenant.",
			},
			"ram_used_mb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The amount of RAM currently consumed by the tenant, in MB.",
			},
			"storage_used_gb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The amount of storage currently consumed by the tenant, in GB.",
			},
		},
	}
}

func publicCloudVMQuotaRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	quota, err := c.PublicCloudVM().Quota().Read(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_quota")
	for k, v := range helpers.FlattenPublicCloudVMQuota(quota) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
