package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasHosts() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all hosts from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSHostsRead,

		Schema: map[string]*schema.Schema{
			// In
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter hosts by machine manager ID.",
			},
			"pool_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter hosts by pool ID.",
			},

			// Out
			"hosts": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of hosts matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the host.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the host.",
						},
						"internal_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internal identifier of the host in the Open IaaS system.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this host belongs to.",
						},
						"machine_manager_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the machine manager this host belongs to.",
						},
						"machine_manager_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the machine manager this host belongs to.",
						},
						"pool": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the pool this host belongs to.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the pool.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the pool.",
									},
									"type": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Information about the pool type.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The key identifier of the pool type.",
												},
												"description": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The description of the pool type.",
												},
											},
										},
									},
								},
							},
						},
						"master": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether this host is a master node in the pool.",
						},
						"uptime": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The uptime of the host in seconds.",
						},
						"power_state": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The current power state of the host.",
						},
						"update_data": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the update status of the host.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"maintenance_mode": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether the host is in maintenance mode.",
									},
									"status": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The current update status of the host.",
									},
								},
							},
						},
						"reboot_required": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether a reboot is required for the host.",
						},
						"virtual_machines": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of virtual machine IDs running on this host.",

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"metrics": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Resource metrics for the host.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"xoa": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "XOA hypervisor information.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"version": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The XOA version.",
												},
												"full_name": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The full name of the XOA version.",
												},
												"build": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The XOA build number.",
												},
											},
										},
									},
									"memory": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Memory metrics for the host.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"usage": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The memory usage in bytes.",
												},
												"size": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The total memory size in bytes.",
												},
											},
										},
									},
									"cpu": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "CPU metrics for the host.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"sockets": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The number of CPU sockets.",
												},
												"cores": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The number of CPU cores.",
												},
												"model": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The CPU model identifier.",
												},
												"model_name": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The CPU model name.",
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
