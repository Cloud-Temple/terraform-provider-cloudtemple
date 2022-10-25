package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceVirtualMachineRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_type": {
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
			"datastore_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"consolidation_needed": {
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
			"virtual_datacenter_id": {
				Type:     schema.TypeString,
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

				Elem: &schema.Schema{
					Type: schema.TypeString,
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
					},
				},
			},
		},
	}
}

func dataSourceVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	vm, err := client.Compute().VirtualMachine().Read(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, vm.ID)
	sw.set("name", vm.Name)
	sw.set("moref", vm.Moref)
	sw.set("machine_manager_type", vm.MachineManagerType)
	sw.set("machine_manager_id", vm.MachineManagerId)
	sw.set("machine_manager_name", vm.MachineManagerName)
	sw.set("datastore_name", vm.DatastoreName)
	sw.set("consolidation_needed", vm.ConsolidationNeeded)
	sw.set("template", vm.Template)
	sw.set("power_state", vm.PowerState)
	sw.set("hardware_version", vm.HardwareVersion)
	sw.set("num_cores_per_socket", vm.NumCoresPerSocket)
	sw.set("operating_system_name", vm.OperatingSystemName)
	sw.set("cpu", vm.Cpu)
	sw.set("cpu_hot_add_enabled", vm.CpuHotAddEnabled)
	sw.set("cpu_hot_remove_enabled", vm.CpuHotRemoveEnabled)
	sw.set("memory_hot_add_enabled", vm.MemoryHotAddEnabled)
	sw.set("memory", vm.Memory)
	sw.set("cpu_usage", vm.CpuUsage)
	sw.set("memory_usage", vm.MemoryUsage)
	sw.set("tools", vm.Tools)
	sw.set("tools_version", vm.ToolsVersion)
	sw.set("virtual_datacenter_id", vm.VirtualDatacenterId)
	sw.set("distributed_virtual_port_group_ids", vm.DistributedVirtualPortGroupIds)
	sw.set("spp_mode", vm.SppMode)
	sw.set("snapshoted", vm.Snapshoted)
	sw.set("triggered_alarms", vm.TriggeredAlarms)

	disk := make([]interface{}, len(vm.ReplicationConfig.Disk))
	for i, d := range vm.ReplicationConfig.Disk {
		disk[i] = map[string]interface{}{
			"key":                 d.Key,
			"disk_replication_id": d.DiskReplicationId,
		}
	}
	sw.set("replication_config", []interface{}{
		map[string]interface{}{
			"generation":              vm.ReplicationConfig.Generation,
			"vm_replication_id":       vm.ReplicationConfig.VmReplicationId,
			"rpo":                     vm.ReplicationConfig.Rpo,
			"quiesce_guest_enabled":   vm.ReplicationConfig.QuiesceGuestEnabled,
			"paused":                  vm.ReplicationConfig.Paused,
			"opp_updates_enabled":     vm.ReplicationConfig.OppUpdatesEnabled,
			"net_compression_enabled": vm.ReplicationConfig.NetCompressionEnabled,
			"net_encryption_enabled":  vm.ReplicationConfig.NetEncryptionEnabled,
			"encryption_destination":  vm.ReplicationConfig.EncryptionDestination,
			"disk":                    disk,
		},
	})

	extraConfig := make([]interface{}, len(vm.ExtraConfig))
	for i, ec := range vm.ExtraConfig {
		extraConfig[i] = map[string]interface{}{
			"key":   ec.Key,
			"value": ec.Value,
		}
	}
	sw.set("extra_config", extraConfig)
	sw.set("storage", []interface{}{
		map[string]interface{}{
			"committed":   vm.Storage.Committed,
			"uncommitted": vm.Storage.Uncommitted,
		},
	})
	sw.set("boot_options", []interface{}{
		map[string]interface{}{
			"firmware":           vm.BootOptions.Firmware,
			"boot_delay":         vm.BootOptions.BootDelay,
			"enter_bios_setup":   vm.BootOptions.EnterBIOSSetup,
			"boot_retry_enabled": vm.BootOptions.BootRetryEnabled,
			"boot_retry_delay":   vm.BootOptions.BootRetryDelay,
		},
	})

	return sw.diags
}
