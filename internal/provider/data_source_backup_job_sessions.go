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
		Description: "Provides information about backup job sessions.",

		ReadContext: backupJobSessionsRead,

		Schema: map[string]*schema.Schema{
			// Out
			"job_sessions": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of backup job sessions.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the job session.",
						},
						"job_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the backup job.",
						},
						"sla_policy_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of SLA policy associated with the job.",
						},
						"job_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the backup job.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the job session (e.g., backup, restore).",
						},
						"duration": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The duration of the job session in milliseconds.",
						},
						"start": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The start time of the job session as a Unix timestamp.",
						},
						"end": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The end time of the job session as a Unix timestamp.",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The current status of the job session (e.g., RESOURCE ACTIVE, FAILED, PARTIAL, COMPLETED, ...).",
						},
						"statistics": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Statistics about the job session execution.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"total": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The total number of operations in the job session.",
									},
									"success": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of successful operations in the job session.",
									},
									"failed": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of failed operations in the job session.",
									},
									"skipped": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of skipped operations in the job session.",
									},
								},
							},
						},
						"sla_policies": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of SLA policies associated with the job session.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The unique identifier of the SLA policy.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the SLA policy.",
									},
									"href": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The hyperlink reference to the SLA policy.",
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
