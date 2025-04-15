package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenIaasTemplates() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all templates from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSTemplatesRead,

		Schema: map[string]*schema.Schema{
			// In
			"machine_manager_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter templates by machine manager ID.",
			},
			"pool_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter templates by pool ID.",
			},

			// Out
			"templates": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of templates matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the template.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the template.",
						},
						"internal_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internal identifier of the template in the Open IaaS system.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this template belongs to.",
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
							Description: "The current power state of the template (e.g., Running, Halted, Paused, etc...).",
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
				},
			},
		},
	}
}

// computeOpenIaaSTemplatesRead lit les templates OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSTemplatesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les templates OpenIaaS
	templates, err := c.Compute().OpenIaaS().Template().List(ctx, &client.OpenIaaSTemplateFilter{
		MachineManagerId: d.Get("machine_manager_id").(string),
		PoolId:           d.Get("pool_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_templates")

	// Mapper manuellement les données en utilisant la fonction helper
	tfTemplates := make([]map[string]interface{}, len(templates))
	for i, template := range templates {
		tfTemplates[i] = helpers.FlattenOpenIaaSTemplate(template)
	}

	// Définir les données dans le state
	if err := d.Set("templates", tfTemplates); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
