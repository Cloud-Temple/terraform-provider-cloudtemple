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

func dataSourceBackupSLAPolicy() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific backup SLA policy.",

		ReadContext: backupSLAPolicyRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the SLA policy to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the SLA policy to retrieve. Conflicts with `id`.",
			},

			// Out
			"sub_policies": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of sub-policies contained within this SLA policy.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the sub-policy.",
						},
						"use_encryption": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Indicates whether encryption is used for this sub-policy.",
						},
						"software": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Indicates whether this is a software-based sub-policy.",
						},
						"site": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The site associated with this sub-policy.",
						},
						"retention": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Retention settings for this sub-policy.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"age": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The retention age in days for backups created by this sub-policy.",
									},
								},
							},
						},
						"trigger": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Trigger settings for this sub-policy.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"frequency": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The frequency of the trigger",
									},
									"type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The type of the trigger. (eg. SUBHOURLY, HOURLY, DAILY, WEEKLY, MONTHLY)",
									},
									"activate_date": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The activation date of the trigger as a Unix timestamp.",
									},
								},
							},
						},
						"target": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Target settings for this sub-policy.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the target resource.",
									},
									"href": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The href (URL) of the target resource.",
									},
									"resource_type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The type of the target resource.",
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

// backupSLAPolicyRead lit une politique SLA de backup et la mappe dans le state Terraform
func backupSLAPolicyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var policy *client.BackupSLAPolicy
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		policies, err := c.Backup().SLAPolicy().List(ctx, &client.BackupSLAPolicyFilter{})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find SLA policy named %q: %s", name, err))
		}
		for _, p := range policies {
			if p.Name == name {
				policy = p
				break
			}
		}
		if policy == nil {
			return diag.FromErr(fmt.Errorf("failed to find SLA policy named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			policy, err = c.Backup().SLAPolicy().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if policy == nil {
				return diag.FromErr(fmt.Errorf("failed to find SLA policy with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(policy.ID)

	// Mapper les données en utilisant la fonction helper
	policyData := helpers.FlattenBackupSLAPolicy(policy)

	// Définir les données dans le state
	for k, v := range policyData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
