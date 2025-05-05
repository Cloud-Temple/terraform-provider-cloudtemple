package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupJob() *schema.Resource {
	return &schema.Resource{
		Description: "Provides information about a specific backup job.",

		ReadContext: backupJobRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  IsNumber,
				Description:   "The ID of the backup job. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				Description:   "The name of the backup job. Conflicts with `id`.",
			},

			// Out
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
	}
}

// backupJobRead lit un job de backup et le mappe dans le state Terraform
func backupJobRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var job *client.BackupJob
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		jobs, err := c.Backup().Job().List(ctx, nil)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find job named %q: %s", name, err))
		}
		for _, j := range jobs {
			if j.Name == name {
				job = j
				break
			}
		}
		if job == nil {
			return diag.FromErr(fmt.Errorf("failed to find job named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			job, err = c.Backup().Job().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if job == nil {
				return diag.FromErr(fmt.Errorf("failed to find job with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(job.ID)

	// Mapper les données en utilisant la fonction helper
	jobData := helpers.FlattenBackupJob(job)

	// Définir les données dans le state
	for k, v := range jobData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
