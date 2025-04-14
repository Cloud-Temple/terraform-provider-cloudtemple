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
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"pool_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			// Out
			"templates": {
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
						"machine_manager_id": {
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
