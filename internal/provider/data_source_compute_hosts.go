package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceHosts() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of hosts.",

		ReadContext: computeHostsRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter hosts by name.",
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter hosts by machine manager ID.",
			},
			"datacenter_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter hosts by datacenter ID.",
			},
			"datastore_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter hosts by datastore ID.",
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter hosts by host cluster ID.",
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
						"moref": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The managed object reference ID of the host.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this host belongs to.",
						},
						"metrics": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Resource metrics for the host.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"esx": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "ESX hypervisor information.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"version": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The ESX version.",
												},
												"build": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The ESX build number.",
												},
												"full_name": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The full name of the ESX version.",
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
												"overall_cpu_usage": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The overall CPU usage in MHz.",
												},
												"cpu_mhz": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The CPU frequency in MHz.",
												},
												"cpu_cores": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The number of CPU cores.",
												},
												"cpu_threads": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The number of CPU threads.",
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
												"memory_size": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The total memory size in bytes.",
												},
												"memory_usage": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The memory usage in bytes.",
												},
											},
										},
									},
									"maintenance_status": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether the host is in maintenance mode.",
									},
									"uptime": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The host uptime in seconds.",
									},
									"connected": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether the host is connected.",
									},
								},
							},
						},
						"virtual_machines": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of virtual machines running on this host.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the virtual machine.",
									},
									"type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The type of the virtual machine.",
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

// computeHostsRead lit les hôtes et les mappe dans le state Terraform
func computeHostsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les hôtes
	hosts, err := c.Compute().Host().List(ctx, &client.HostFilter{
		Name:             d.Get("name").(string),
		MachineManagerID: d.Get("machine_manager_id").(string),
		DatacenterID:     d.Get("datacenter_id").(string),
		HostClusterID:    d.Get("host_cluster_id").(string),
		DatastoreID:      d.Get("datastore_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("hosts")

	// Mapper manuellement les données en utilisant la fonction helper
	tfHosts := make([]map[string]interface{}, len(hosts))
	for i, host := range hosts {
		tfHosts[i] = helpers.FlattenHost(host)
	}

	// Définir les données dans le state
	if err := d.Set("hosts", tfHosts); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
