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
				Description:   "The ID of the template to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the template to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The ID of the machine manager to filter templates by. Required when searching by `name`.",
			},
			"pool_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				Description:   "The ID of the pool to filter templates by.",
			},

			// Out
			"machine_manager_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the machine manager this template belongs to.",
			},
			"machine_manager_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the machine manager (e.g., XenServer, VMware).",
			},
			"internal_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internal identifier of the template in the Open IaaS system.",
			},
			"cpu": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of virtual CPUs in the template.",
			},
			"num_cores_per_socket": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of cores per CPU socket in the template.",
			},
			"memory": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The amount of memory in Bytes allocated to the template.",
			},
			"power_state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current power state of the template (e.g., Running, Halted, Paused, ...).",
			},
			"snapshots": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of snapshot IDs associated with this template.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"sla_policies": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of SLA policy IDs applied to this template.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"disks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of virtual disks attached to this template.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the virtual disk.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the virtual disk.",
						},
						"size": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The size of the virtual disk in bytes.",
						},
						"storage_repository": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the storage repository where the disk is located.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the storage repository.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the storage repository.",
									},
								},
							},
						},
					},
				},
			},
			"network_adapters": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of network adapters attached to this template.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the network adapter.",
						},
						"mac_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The MAC address of the network adapter.",
						},
						"mtu": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The Maximum Transmission Unit (MTU) size for the network adapter.",
						},
						"attached": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the network adapter is attached to the template.",
						},
						"network": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the network this adapter is connected to.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the network.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the network.",
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
