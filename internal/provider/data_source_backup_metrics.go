package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupMetrics() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			rang := d.Get("range").(int)

			coverage, err := client.Backup().Metrics().Coverage(ctx)
			if err != nil {
				return nil, err
			}
			history, err := client.Backup().Metrics().History(ctx, rang)
			if err != nil {
				return nil, err
			}
			platform, err := client.Backup().Metrics().Platform(ctx)
			if err != nil {
				return nil, err
			}
			platformCPU, err := client.Backup().Metrics().PlatformCPU(ctx)
			if err != nil {
				return nil, err
			}
			policies, err := client.Backup().Metrics().Policies(ctx)
			if err != nil {
				return nil, err
			}
			virtualMachines, err := client.Backup().Metrics().VirtualMachines(ctx)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"id":               "job_sessions",
				"coverage":         *coverage,
				"history":          *history,
				"platform":         *platform,
				"platform_cpu":     *platformCPU,
				"policies":         policies,
				"virtual_machines": *virtualMachines,
			}, nil
		}),

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
