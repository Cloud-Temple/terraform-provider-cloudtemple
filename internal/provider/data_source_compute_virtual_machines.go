package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualMachines() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all virtual machines from a vCenter infrastructure.",

		ReadContext: dataSourceVirtualMachinesRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"datacenters": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"networks": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"datastores": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"hosts": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"host_clusters": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			// Out
			"virtual_machines": {
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
						"moref": {
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
						"datacenter_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"host_cluster_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"datastore_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"datastore_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"datastore_cluster_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"consolidation_needed": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"expose_hardware_virtualization": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"template": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"power_state": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"hardware_version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"num_cores_per_socket": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"operating_system_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"guest_operating_system_moref": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"cpu": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"cpu_hot_add_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"cpu_hot_remove_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"memory_hot_add_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"memory": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"cpu_usage": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"memory_usage": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"tools": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"tools_version": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"distributed_virtual_port_group_ids": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"spp_mode": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"snapshoted": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"triggered_alarms": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"status": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"replication_config": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"generation": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"vm_replication_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"rpo": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"quiesce_guest_enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"paused": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"opp_updates_enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"net_compression_enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"net_encryption_enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"encryption_destination": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"disk": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"disk_replication_id": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
								},
							},
						},
						"extra_config": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"value": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"storage": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"committed": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"uncommitted": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"boot_options": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"firmware": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"boot_delay": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"enter_bios_setup": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"boot_retry_enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"boot_retry_delay": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"efi_secure_boot_enabled": {
										Type:     schema.TypeBool,
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

// dataSourceVirtualMachinesRead lit les machines virtuelles et les mappe dans le state Terraform
func dataSourceVirtualMachinesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les machines virtuelles
	virtualMachines, err := c.Compute().VirtualMachine().List(ctx, &client.VirtualMachineFilter{
		Name:             d.Get("name").(string),
		MachineManagerID: d.Get("machine_manager_id").(string),
		Datacenters:      GetStringList(d, "datacenters"),
		Networks:         GetStringList(d, "networks"),
		Datastores:       GetStringList(d, "datastores"),
		Hosts:            GetStringList(d, "hosts"),
		HostClusters:     GetStringList(d, "host_clusters"),
		AllOptions:       true,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("virtual_machines")

	// Mapper manuellement les données en utilisant la fonction helper
	tfVirtualMachines := make([]map[string]interface{}, len(virtualMachines))
	for i, vm := range virtualMachines {
		tfVirtualMachines[i] = helpers.FlattenVirtualMachine(vm)
		tfVirtualMachines[i]["id"] = vm.ID
	}

	// Définir les données dans le state
	if err := d.Set("virtual_machines", tfVirtualMachines); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
