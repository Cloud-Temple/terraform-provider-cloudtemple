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
		Description: "",

		ReadContext: backupMetricsRead,

		Schema: map[string]*schema.Schema{
			"range": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  4,
			},

			// Out
			"coverage": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"failed_resources": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"protected_resources": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"unprotected_resources": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"total_resources": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"history": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"total_runs": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"sucess_percent": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"failed": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"warning": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"success": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"running": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"platform": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"build": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"date": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"product": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"epoch": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"deployment_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"platform_cpu": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cpu_util": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"policies": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"trigger_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"number_of_protected_vm": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"virtual_machines": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"in_spp": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"in_compute": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"with_backup": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"in_sla": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"in_offloading_sla": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"tsm_offloading_factor": {
							Type:     schema.TypeInt,
							Computed: true,
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
