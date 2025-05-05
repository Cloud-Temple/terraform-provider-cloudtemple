package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasBackups() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of backups from an Open IaaS infrastructure.",

		ReadContext: backupOpenIaasBackupsRead,

		Schema: map[string]*schema.Schema{
			// In
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter backups by machine manager ID.",
			},
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter backups by virtual machine ID.",
			},
			"deleted": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Include backups of deleted virtual machines when set to true.",
			},

			// Out
			"backups": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of backups matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsUUID,
							Description:  "The unique identifier of the backup.",
						},
						"internal_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internal identifier of the backup in the Open IaaS system.",
						},
						"mode": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The backup mode (e.g., full, incremental).",
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
				},
			},
		},
	}
}

// backupOpenIaasBackupsRead lit les backups OpenIaaS et les mappe dans le state Terraform
func backupOpenIaasBackupsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les backups
	backups, err := c.Backup().OpenIaaS().Backup().List(ctx, &client.OpenIaasBackupFilter{
		MachineManagerId: d.Get("machine_manager_id").(string),
		VirtualMachineId: d.Get("virtual_machine_id").(string),
		Deleted:          d.Get("deleted").(bool),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("backups")

	// Mapper manuellement les données en utilisant la fonction helper
	tfBackups := make([]map[string]interface{}, len(backups))
	for i, backup := range backups {
		tfBackups[i] = helpers.FlattenBackupOpenIaasBackup(backup)
	}

	// Définir les données dans le state
	if err := d.Set("backups", tfBackups); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
