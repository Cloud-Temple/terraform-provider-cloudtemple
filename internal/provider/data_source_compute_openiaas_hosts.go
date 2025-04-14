package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenIaasHosts() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all hosts from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSHostsRead,

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
			"hosts": {
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
						"machine_manager_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"pool": {
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
									"type": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"description": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
								},
							},
						},
						"master": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"uptime": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"power_state": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"update_data": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"maintenance_mode": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"status": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"reboot_required": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"virtual_machines": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"metrics": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"xoa": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"version": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"full_name": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"build": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
									"memory": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"usage": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"size": {
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
									},
									"cpu": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"sockets": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"cores": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"model": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"model_name": {
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

// computeOpenIaaSHostsRead lit les hôtes OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSHostsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les hôtes OpenIaaS
	hosts, err := c.Compute().OpenIaaS().Host().List(ctx, &client.OpenIaasHostFilter{
		MachineManagerId: d.Get("machine_manager_id").(string),
		PoolId:           d.Get("pool_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_hosts")

	// Mapper manuellement les données en utilisant la fonction helper
	tfHosts := make([]map[string]interface{}, len(hosts))
	for i, host := range hosts {
		tfHosts[i] = helpers.FlattenOpenIaaSHost(host)
	}

	// Définir les données dans le state
	if err := d.Set("hosts", tfHosts); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
