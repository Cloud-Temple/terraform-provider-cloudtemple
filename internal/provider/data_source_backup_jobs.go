package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupJobs() *schema.Resource {
	return &schema.Resource{
		Description: "Provides information about backup jobs.",

		ReadContext: backupJobsRead,

		Schema: map[string]*schema.Schema{
			// Out
			"jobs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of backup jobs.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the backup job.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the backup job.",
						},
						"display_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The display name of the backup job.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the backup job.",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The current status of the backup job.",
						},
						"policy_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the SLA policy associated with the backup job.",
						},
					},
				},
			},
		},
	}
}

// backupJobsRead lit les jobs de backup et les mappe dans le state Terraform
func backupJobsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les données
	jobs, err := c.Backup().Job().List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("jobs")

	// Mapper manuellement les données en utilisant la fonction helper
	tfJobs := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		tfJobs[i] = helpers.FlattenBackupJob(job)
	}

	// Définir les données dans le state
	if err := d.Set("jobs", tfJobs); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
