package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasBackupPolicies() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of backup policies from an Open IaaS infrastructure.",

		ReadContext: backupOpenIaasPoliciesRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter policies by name.",
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter policies by machine manager ID.",
			},
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter policies by virtual machine ID.",
			},
			// Out
			"policies": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of backup policies matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the backup policy.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the backup policy.",
						},
						"internal_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internal identifier of the policy in the Open IaaS system.",
						},
						"running": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Indicates whether the policy is currently running.",
						},
						"mode": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The backup mode of the policy",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager associated with this policy.",
						},
						"machine_manager_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the machine manager associated with this policy.",
						},
						"virtual_machines": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of virtual machines associated with this backup policy.",
							Elem:        schema.TypeString,
						},
						"schedulers": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of schedulers configured for this backup policy.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"temporarily_disabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Indicates whether the scheduler is temporarily disabled.",
									},
									"retention": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The retention period for backups created by this scheduler (in days).",
									},
									"cron": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The cron expression defining the schedule.",
									},
									"timezone": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The timezone used for the scheduler.",
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

// backupOpenIaasPoliciesRead lit les politiques de backup OpenIaaS et les mappe dans le state Terraform
func backupOpenIaasPoliciesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les politiques
	policies, err := c.Backup().OpenIaaS().Policy().List(ctx, &client.BackupOpenIaasPolicyFilter{
		Name:             d.Get("name").(string),
		MachineManagerId: d.Get("machine_manager_id").(string),
		VirtualMachineId: d.Get("virtual_machine_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("policies")

	// Mapper manuellement les données en utilisant la fonction helper
	tfPolicies := make([]map[string]interface{}, len(policies))
	for i, policy := range policies {
		tfPolicies[i] = helpers.FlattenBackupOpenIaasPolicy(policy)
	}

	// Définir les données dans le state
	if err := d.Set("policies", tfPolicies); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
