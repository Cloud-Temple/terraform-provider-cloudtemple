package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenIaasBackupPolicies() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of backup policies from an Open IaaS infrastructure.",

		ReadContext: backupOpenIaasPoliciesRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"virtual_machine_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			// Out
			"policies": {
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
						"internal_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"running": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"mode": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"schedulers": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"temporarily_disabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"retention": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"cron": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"timezone": {
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
