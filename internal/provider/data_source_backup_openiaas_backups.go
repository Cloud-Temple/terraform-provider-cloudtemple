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
			},
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"deleted": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			// Out
			"backups": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsUUID,
						},
						"internal_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"mode": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_machine": {
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
								},
							},
						},
						"policy": {
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
								},
							},
						},
						"is_virtual_machine_deleted": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"timestamp": {
							Type:     schema.TypeInt,
							Computed: true,
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
