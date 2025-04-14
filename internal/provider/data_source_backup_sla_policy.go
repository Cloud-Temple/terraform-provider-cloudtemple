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
		Description: "",

		ReadContext: backupSLAPolicyRead,

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

			// Out
			"sub_policies": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"use_encryption": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"software": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"site": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"retention": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"age": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"trigger": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"frequency": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"type": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"activate_date": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"target": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"href": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"resource_type": {
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
