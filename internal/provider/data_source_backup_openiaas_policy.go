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

func dataSourceOpenIaasBackupPolicy() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific backup policy from an Open IaaS infrastructure.",

		ReadContext: backupOpenIaasPolicyRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name"},
			},

			// Out
			"machine_manager_name": {
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
	}
}

// backupOpenIaasPolicyRead lit une politique de backup OpenIaaS et la mappe dans le state Terraform
func backupOpenIaasPolicyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var policy *client.BackupOpenIaasPolicy
	var err error

	// Recherche par ID
	id := d.Get("id").(string)
	if id != "" {
		policy, err = c.Backup().OpenIaaS().Policy().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if policy == nil {
			return diag.FromErr(fmt.Errorf("failed to find backup policy with id %q", id))
		}
	} else {
		// Recherche par nom
		name := d.Get("name").(string)
		if name != "" {
			policies, err := c.Backup().OpenIaaS().Policy().List(ctx, &client.BackupOpenIaasPolicyFilter{
				Name:             name,
				MachineManagerId: d.Get("machine_manager_id").(string),
			})
			if err != nil {
				return diag.FromErr(fmt.Errorf("failed to list backup policies: %s", err))
			}
			for _, p := range policies {
				if p.Name == name {
					policy = p
					break
				}
			}
			if policy == nil {
				return diag.FromErr(fmt.Errorf("failed to find backup policy named %q", name))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(policy.ID)

	// Mapper les données en utilisant la fonction helper
	policyData := helpers.FlattenBackupOpenIaasPolicy(policy)

	// Définir les données dans le state
	for k, v := range policyData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
