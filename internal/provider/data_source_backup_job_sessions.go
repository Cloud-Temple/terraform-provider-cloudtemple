package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupJobSessions() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			jobSessions, err := client.Backup().JobSession().List(ctx, nil)
			return map[string]interface{}{
				"id":           "job_sessions",
				"job_sessions": jobSessions,
			}, err
		}),

		Schema: map[string]*schema.Schema{
			// Out
			"job_sessions": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"job_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"sla_policy_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"job_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"duration": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"start": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"end": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"statistics": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"total": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"success": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"failed": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"skipped": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"sla_policies": {
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
									"href": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
