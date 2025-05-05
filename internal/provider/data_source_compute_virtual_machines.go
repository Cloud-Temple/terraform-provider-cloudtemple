package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVirtualMachines() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all virtual machines from a vCenter infrastructure.",

		ReadContext: dataSourceVirtualMachinesRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter virtual machines by name.",
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter virtual machines by machine manager (vCenter) ID.",
			},
			"datacenters": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Filter virtual machines by datacenter IDs.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"networks": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Filter virtual machines by network IDs.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"datastores": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Filter virtual machines by datastore IDs.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"hosts": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Filter virtual machines by host IDs.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"host_clusters": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Filter virtual machines by host cluster IDs.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			// Out
			"virtual_machines": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of virtual machines matching the specified filters.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the virtual machine.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the virtual machine.",
						},
						"moref": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The managed object reference ID of the virtual machine in the hypervisor.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager (vCenter) where the virtual machine is located.",
						},
						"machine_manager_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the machine manager (vCenter) where the virtual machine is located.",
						},
						"datacenter_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the datacenter where the virtual machine is located.",
						},
						"host_cluster_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the host cluster where the virtual machine is running.",
						},
						"datastore_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the datastore where the virtual machine is stored.",
						},
						"datastore_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the datastore where the virtual machine is stored.",
						},
						"datastore_cluster_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the datastore cluster where the virtual machine is stored.",
						},
						"consolidation_needed": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the virtual machine needs consolidation.",
						},
						"expose_hardware_virtualization": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether hardware virtualization is exposed to the guest operating system.",
						},
						"template": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the virtual machine is a template.",
						},
						"power_state": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The power state of the virtual machine",
						},
						"hardware_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The hardware version of the virtual machine.",
						},
						"num_cores_per_socket": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of cores per socket in the virtual machine.",
						},
						"operating_system_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the operating system running on the virtual machine.",
						},
						"guest_operating_system_moref": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The managed object reference ID of the guest operating system in the hypervisor.",
						},
						"cpu": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of virtual CPUs allocated to the virtual machine.",
						},
						"cpu_hot_add_enabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether CPU hot add is enabled for the virtual machine.",
						},
						"cpu_hot_remove_enabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether CPU hot remove is enabled for the virtual machine.",
						},
						"memory_hot_add_enabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether memory hot add is enabled for the virtual machine.",
						},
						"memory": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The amount of memory allocated to the virtual machine in Bytes.",
						},
						"cpu_usage": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The current CPU usage of the virtual machine in MHz.",
						},
						"memory_usage": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The current memory usage of the virtual machine in Bytes.",
						},
						"tools": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The status of VMware Tools in the virtual machine.",
						},
						"tools_version": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The version of VMware Tools installed in the virtual machine.",
						},
						"distributed_virtual_port_group_ids": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of distributed virtual port group IDs associated with the virtual machine.",

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"spp_mode": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The SPP (Storage Policy Protection) mode of the virtual machine.",
						},
						"snapshoted": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the virtual machine has snapshots.",
						},
						"triggered_alarms": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of alarms that have been triggered for this virtual machine.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the triggered alarm.",
									},
									"status": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The status of the triggered alarm.",
									},
								},
							},
						},
						"replication_config": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Configuration for virtual machine replication.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"generation": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The generation number of the replication configuration.",
									},
									"vm_replication_id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the virtual machine replication.",
									},
									"rpo": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Recovery Point Objective (RPO) in minutes.",
									},
									"quiesce_guest_enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether guest quiescing is enabled during replication.",
									},
									"paused": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether replication is paused.",
									},
									"opp_updates_enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether opportunistic updates are enabled.",
									},
									"net_compression_enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether network compression is enabled for replication.",
									},
									"net_encryption_enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether network encryption is enabled for replication.",
									},
									"encryption_destination": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether encryption is enabled at the destination.",
									},
									"disk": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "List of disks included in the replication.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The key identifier of the disk.",
												},
												"disk_replication_id": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The ID of the disk replication.",
												},
											},
										},
									},
								},
							},
						},
						"extra_config": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Extra configuration parameters for the virtual machine.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The key of the configuration parameter.",
									},
									"value": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The value of the configuration parameter.",
									},
								},
							},
						},
						"storage": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Storage usage information for the virtual machine.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"committed": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The amount of storage space committed to the virtual machine in bytes.",
									},
									"uncommitted": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The amount of storage space that can potentially be used by the virtual machine in bytes.",
									},
								},
							},
						},
						"boot_options": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Boot configuration options for the virtual machine.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"firmware": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The firmware type used by the virtual machine (e.g., BIOS, EFI).",
									},
									"boot_delay": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The delay in milliseconds before booting the virtual machine.",
									},
									"enter_bios_setup": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether the virtual machine enters BIOS setup on next boot.",
									},
									"boot_retry_enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether boot retry is enabled for the virtual machine.",
									},
									"boot_retry_delay": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The delay in milliseconds before retrying to boot the virtual machine.",
									},
									"efi_secure_boot_enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether EFI secure boot is enabled for the virtual machine.",
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
