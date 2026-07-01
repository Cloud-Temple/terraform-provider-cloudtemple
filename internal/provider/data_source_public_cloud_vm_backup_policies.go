package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePublicCloudVMBackupPolicies() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all Public Cloud VM Instances backup policies of the tenant.",

		ReadContext: publicCloudVMBackupPoliciesRead,

		Schema: map[string]*schema.Schema{
			// Out
			"backup_policies": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of backup policies.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the backup policy.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the backup policy.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The human-readable description of the backup policy.",
						},
						"retention": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of restore points retained by the policy.",
						},
						"schedule_cron": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The cron expression driving the backup schedule.",
						},
						"schedule_window_start_hour": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The hour of day (0-23) at which the backup window starts.",
						},
						"schedule_window_duration_minutes": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The duration (minutes) of the backup window.",
						},
					},
				},
			},
		},
	}
}

func publicCloudVMBackupPoliciesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	policies, err := c.PublicCloudVM().BackupPolicy().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_backup_policies")

	tfPolicies := make([]map[string]interface{}, len(policies))
	for i, p := range policies {
		tfPolicies[i] = helpers.FlattenPublicCloudVMBackupPolicy(p)
	}

	if err := d.Set("backup_policies", tfPolicies); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
