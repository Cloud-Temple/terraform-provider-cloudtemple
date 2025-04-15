package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupMetrics() *schema.Resource {
	return &schema.Resource{
		Description: "Provides metrics and statistics about the backup system.",

		ReadContext: backupMetricsRead,

		Schema: map[string]*schema.Schema{
			"range": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     4,
				Description: "The time range in days for which to retrieve metrics. Defaults to 4 days.",
			},

			// Out
			"coverage": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Statistics about resource protection coverage.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"failed_resources": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of resources with failed protection.",
						},
						"protected_resources": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of successfully protected resources.",
						},
						"unprotected_resources": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of resources without protection.",
						},
						"total_resources": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The total number of resources in the system.",
						},
					},
				},
			},
			"history": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Historical statistics about backup job runs.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"total_runs": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The total number of backup job runs in the history period.",
						},
						"sucess_percent": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The percentage of successful backup job runs.",
						},
						"failed": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of failed backup job runs.",
						},
						"warning": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of backup job runs with warnings.",
						},
						"success": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of successful backup job runs.",
						},
						"running": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of currently running backup jobs.",
						},
					},
				},
			},
			"platform": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Information about the backup platform.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The version of the backup platform.",
						},
						"build": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The build number of the backup platform.",
						},
						"date": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The build date of the backup platform.",
						},
						"product": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The product name of the backup platform.",
						},
						"epoch": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The epoch timestamp of the backup platform build.",
						},
						"deployment_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The deployment type of the backup platform.",
						},
					},
				},
			},
			"platform_cpu": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "CPU utilization metrics for the backup platform.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cpu_util": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The CPU utilization percentage of the backup platform.",
						},
					},
				},
			},
			"policies": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Information about backup policies in the system.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the backup policy.",
						},
						"trigger_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The trigger type of the backup policy.",
						},
						"number_of_protected_vm": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of virtual machines protected by this policy.",
						},
					},
				},
			},
			"virtual_machines": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Statistics about virtual machines in the backup system.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"in_spp": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of virtual machines in the SPP backup system.",
						},
						"in_compute": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of virtual machines in the compute system.",
						},
						"with_backup": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of virtual machines with backups.",
						},
						"in_sla": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of virtual machines covered by SLA policies.",
						},
						"in_offloading_sla": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of virtual machines in offloading SLA policies.",
						},
						"tsm_offloading_factor": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The TSM offloading factor for virtual machines.",
						},
					},
				},
			},
		},
	}
}

// backupMetricsRead lit les métriques de backup et les mappe dans le state Terraform
func backupMetricsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer le paramètre range
	rang := d.Get("range").(int)

	// Récupérer les données
	coverage, err := c.Backup().Metrics().Coverage(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	history, err := c.Backup().Metrics().History(ctx, rang)
	if err != nil {
		return diag.FromErr(err)
	}
	platform, err := c.Backup().Metrics().Platform(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	platformCPU, err := c.Backup().Metrics().PlatformCPU(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	policies, err := c.Backup().Metrics().Policies(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	virtualMachines, err := c.Backup().Metrics().VirtualMachines(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("job_sessions")

	// Mapper les données en utilisant les fonctions helper
	if err := d.Set("coverage", []interface{}{helpers.FlattenBackupMetricsCoverage(coverage)}); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("history", []interface{}{helpers.FlattenBackupMetricsHistory(history)}); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("platform", []interface{}{helpers.FlattenBackupMetricsPlatform(platform)}); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("platform_cpu", []interface{}{helpers.FlattenBackupMetricsPlatformCPU(platformCPU)}); err != nil {
		return diag.FromErr(err)
	}

	// Mapper les policies
	tfPolicies := make([]map[string]interface{}, len(policies))
	for i, policy := range policies {
		tfPolicies[i] = helpers.FlattenBackupMetricsPolicy(policy)
	}
	if err := d.Set("policies", tfPolicies); err != nil {
		return diag.FromErr(err)
	}

	// Mapper les virtual machines
	if err := d.Set("virtual_machines", []interface{}{helpers.FlattenBackupMetricsVirtualMachines(virtualMachines)}); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
