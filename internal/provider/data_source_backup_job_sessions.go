package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupJobSessions() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: backupJobSessionsRead,

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

// backupJobSessionsRead lit les sessions de job de backup et les mappe dans le state Terraform
func backupJobSessionsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	// Utilisation explicite du type client.Client pour que l'import soit reconnu
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les données
	jobSessions, err := c.Backup().JobSession().List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("job_sessions")

	// Mapper manuellement les données en utilisant la fonction helper
	tfJobSessions := make([]map[string]interface{}, len(jobSessions))
	for i, js := range jobSessions {
		tfJobSessions[i] = helpers.FlattenBackupJobSession(js)
	}

	// Définir les données dans le state
	if err := d.Set("job_sessions", tfJobSessions); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
