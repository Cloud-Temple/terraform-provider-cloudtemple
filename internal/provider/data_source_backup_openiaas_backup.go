package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasBackup() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific backup from an Open IaaS infrastructure.",

		ReadContext: backupOpenIaasBackupRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the backup to retrieve.",
			},

			// Out
			"internal_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internal identifier of the backup in the Open IaaS system.",
			},
			"mode": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The backup mode.",
			},
			"virtual_machine": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Information about the virtual machine associated with this backup.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the virtual machine.",
						},
					},
				},
			},
			"policy": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Information about the backup policy used for this backup.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the backup policy.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the backup policy.",
						},
					},
				},
			},
			"is_virtual_machine_deleted": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether the associated virtual machine has been deleted.",
			},
			"size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the backup in bytes.",
			},
			"timestamp": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The timestamp when the backup was created (Unix timestamp).",
			},
		},
	}
}

// backupOpenIaasBackupRead lit un backup OpenIaaS et le mappe dans le state Terraform
func backupOpenIaasBackupRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer l'ID du backup
	id := d.Get("id").(string)
	if id == "" {
		return diag.FromErr(fmt.Errorf("id must be specified"))
	}

	// Récupérer le backup
	backup, err := c.Backup().OpenIaaS().Backup().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if backup == nil {
		return diag.FromErr(fmt.Errorf("failed to find backup with id %q", id))
	}

	// Définir l'ID de la datasource
	d.SetId(backup.ID)

	// Mapper les données en utilisant la fonction helper
	backupData := helpers.FlattenBackupOpenIaasBackup(backup)

	// Définir les données dans le state
	for k, v := range backupData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
