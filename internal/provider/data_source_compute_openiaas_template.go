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

func dataSourceOpenIaasTemplate() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific template from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSTemplateRead,

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
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"pool_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
			},

			// Out
			"machine_manager_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"internal_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cpu": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"num_cores_per_socket": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"memory": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"power_state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"snapshots": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"sla_policies": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"disks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"storage_repository": {
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
					},
				},
			},
			"network_adapters": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"mac_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"mtu": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"attached": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"network": {
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
					},
				},
			},
		},
	}
}

// computeOpenIaaSTemplateRead lit un template OpenIaaS et le mappe dans le state Terraform
func computeOpenIaaSTemplateRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var template *client.OpenIaasTemplate
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		templates, err := c.Compute().OpenIaaS().Template().List(ctx, &client.OpenIaaSTemplateFilter{
			MachineManagerId: d.Get("machine_manager_id").(string),
			PoolId:           d.Get("pool_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find template named %q: %s", name, err))
		}
		for _, t := range templates {
			if t.Name == name {
				template = t
				break
			}
		}
		if template == nil {
			return diag.FromErr(fmt.Errorf("failed to find template named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			template, err = c.Compute().OpenIaaS().Template().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if template == nil {
				return diag.FromErr(fmt.Errorf("failed to find template with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(template.ID)

	// Mapper les données en utilisant la fonction helper
	templateData := helpers.FlattenOpenIaaSTemplate(template)

	// Définir les données dans le state
	for k, v := range templateData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
