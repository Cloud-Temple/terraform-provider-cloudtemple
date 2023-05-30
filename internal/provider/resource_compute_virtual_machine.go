package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Description: `Provision a virtual machine. This allows instances to be created, updated, and deleted.

Virtual machines can be created using three different methods:

  - by creating a new instance with ` + "`guest_operating_system_moref`" + `
  - by cloning an existing virtual machine with ` + "`clone_virtual_machine_id`" + `
  - by deploying a content library item with ` + "`content_library_id` and `content_library_item_id`",

		CreateContext: computeVirtualMachineCreate,
		ReadContext:   computeVirtualMachineRead,
		UpdateContext: computeVirtualMachineUpdate,
		DeleteContext: computeVirtualMachineDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"content_library_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the content library to clone from. Conflict with `clone_virtual_machine_id`.",
				Optional:      true,
				ForceNew:      true,
				RequiredWith:  []string{"content_library_item_id"},
				ConflictsWith: []string{"clone_virtual_machine_id"},
				ValidateFunc:  validation.IsUUID,
			},
			"content_library_item_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the content library item to clone. Conflict with `clone_virtual_machine_id`.",
				Optional:      true,
				ForceNew:      true,
				RequiredWith:  []string{"content_library_id"},
				ConflictsWith: []string{"clone_virtual_machine_id"},
				AtLeastOneOf:  []string{"clone_virtual_machine_id", "guest_operating_system_moref", "content_library_item_id"},
				ValidateFunc:  validation.IsUUID,
			},
			"deploy_options": {
				Type:          schema.TypeMap,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"clone_virtual_machine_id"},

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"clone_virtual_machine_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the virtual machine to clone. Conflict with `content_library_item_id`.",
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"content_library_item_id"},
				AtLeastOneOf:  []string{"clone_virtual_machine_id", "guest_operating_system_moref", "content_library_item_id"},
				ValidateFunc:  validation.IsUUID,
			},
			"guest_operating_system_moref": {
				Type:         schema.TypeString,
				Description:  "The operating system to launch the virtual machine with.",
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{"clone_virtual_machine_id", "guest_operating_system_moref", "content_library_item_id"},
			},
			"datacenter_id": {
				Type:         schema.TypeString,
				Description:  "The datacenter to start the virtual machine in.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_id": {
				Type:         schema.TypeString,
				Description:  "The host to start the virtual machine on.",
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
				Description:  "The host cluster to start the virtual machine on.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"datastore_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"datastore_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"memory": {
				Type:        schema.TypeInt,
				Description: "The quantity of memory to start the virtual machine with.",
				Optional:    true,
				Default:     33554432,
			},
			"cpu": {
				Type:        schema.TypeInt,
				Description: "The number of CPUs to start the virtual machine with.",
				Optional:    true,
				Default:     1,
			},
			"num_cores_per_socket": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			"cpu_hot_add_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"cpu_hot_remove_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"memory_hot_add_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"power_state": {
				Type:         schema.TypeString,
				Description:  "Whether to start the virtual machine.",
				Optional:     true,
				Default:      "off",
				ValidateFunc: validation.StringInSlice([]string{"on", "off"}, false),
			},
			"tags": {
				Type:        schema.TypeMap,
				Description: "The tags to attach to the virtual machine.",
				Optional:    true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"backup_sla_policies": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The IDs of the SLA policies to assign to the virtual machine.",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
			},
			"os_disk": {
				Type:        schema.TypeList,
				Description: "OS disks created from content lib item deployment or virtual machine clone.",
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// In
						"capacity": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"disk_mode": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						// Out
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"disk_unit_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"controller_bus_number": {
							Type:     schema.TypeInt,
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
						"instant_access": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"native_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"disk_path": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"provisioning_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"editable": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},

			// Out
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
			"hardware_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"operating_system_name": {
				Type:     schema.TypeString,
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
		},
	}
}

func computeVirtualMachineCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	name := d.Get("name").(string)

	var activityId string
	var err error
	var fromScratch bool
	cloneVirtualMachineId := d.Get("clone_virtual_machine_id").(string)
	contentLibraryItemId := d.Get("content_library_item_id").(string)

	if cloneVirtualMachineId != "" {
		activityId, err = c.Compute().VirtualMachine().Clone(ctx, &client.CloneVirtualMachineRequest{
			Name:              name,
			VirtualMachineId:  cloneVirtualMachineId,
			PowerOn:           d.Get("power_state").(string) == "on",
			DatacenterId:      d.Get("datacenter_id").(string),
			HostClusterId:     d.Get("host_cluster_id").(string),
			HostId:            d.Get("host_id").(string),
			DatatoreClusterId: d.Get("datastore_cluster_id").(string),
			DatastoreId:       d.Get("datastore_id").(string),
		})
		if err != nil {
			return diag.Errorf("failed to clone virtual machine: %s", err)
		}

		activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		setIdFromActivityState(d, activity)
		if err != nil {
			return diag.Errorf("failed to clone virtual machine, %s", err)
		}

	} else if contentLibraryItemId != "" {
		datastoreId := d.Get("datastore_id").(string)
		if datastoreId == "" {
			return diag.Errorf("'datastore_id' is required when 'content_library_item_id' is used.")
		}

		var deployOptions []*client.DeployOption
		for k, v := range d.Get("deploy_options").(map[string]interface{}) {
			deployOptions = append(deployOptions, &client.DeployOption{
				ID:    k,
				Value: v.(string),
			})
		}

		activityId, err = c.Compute().ContentLibrary().Deploy(ctx, &client.ComputeContentLibraryItemDeployRequest{
			Name:                 name,
			ContentLibraryId:     d.Get("content_library_id").(string),
			ContentLibraryItemId: d.Get("content_library_item_id").(string),
			HostClusterId:        d.Get("host_cluster_id").(string),
			HostId:               d.Get("host_id").(string),
			DatastoreId:          d.Get("datastore_id").(string),
			DatacenterId:         d.Get("datacenter_id").(string),
			PowerOn:              d.Get("power_state").(string) == "on",
			DeployOptions:        deployOptions,
		})
		if err != nil {
			return diag.Errorf("failed to deploy content library item: %s", err)
		}

		activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		setIdFromActivityState(d, activity)
		if err != nil {
			return diag.Errorf("failed to deploy content library item: %s", err)
		}

	} else {
		fromScratch = true
		activityId, err = c.Compute().VirtualMachine().Create(ctx, &client.CreateVirtualMachineRequest{
			Name:                      name,
			DatacenterId:              d.Get("datacenter_id").(string),
			HostId:                    d.Get("host_id").(string),
			HostClusterId:             d.Get("host_cluster_id").(string),
			DatastoreId:               d.Get("datastore_id").(string),
			DatastoreClusterId:        d.Get("datastore_cluster_id").(string),
			Memory:                    d.Get("memory").(int),
			CPU:                       d.Get("cpu").(int),
			GuestOperatingSystemMoref: d.Get("guest_operating_system_moref").(string),
		})
		if err != nil {
			return diag.Errorf("failed to create virtual machine, %s", err)
		}

		activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		setIdFromActivityConcernedItems(d, activity)
		if err != nil {
			return diag.Errorf("failed to create virtual machine: %s", err)
		}
	}

	disks, err := c.Compute().VirtualDisk().List(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to retrieve OS disks: %s", err)
	}

	osDisks := flattenOSDisksData(disks)

	// Overwrite with the desired config
	for i, osDisk := range osDisks {
		if v, ok := d.GetOk(fmt.Sprintf("os_disk.%d", i)); ok {
			vDisk := v.(map[string]interface{})
			disk := osDisk.(map[string]interface{})

			disk["capacity"] = vDisk["capacity"].(int)
			disk["disk_mode"] = vDisk["disk_mode"].(string)
		}
	}

	if err := d.Set("os_disk", osDisks); err != nil {
		return diag.FromErr(err)
	}

	if len(d.Get("backup_sla_policies").(*schema.Set).List()) > 0 {
		// First we need to update the catalog
		jobs, err := c.Backup().Job().List(ctx, &client.BackupJobFilter{
			Type: "catalog",
		})
		if err != nil {
			return diag.Errorf("failed to find catalog job: %s", err)
		}

		var job = &client.BackupJob{}
		for _, currJob := range jobs {
			if currJob.Name == "Hypervisor Inventory" {
				job = currJob
			}
		}

		activityId, err := c.Backup().Job().Run(ctx, &client.BackupJobRunRequest{
			JobId: job.ID,
		})
		if err != nil {
			return diag.Errorf("failed to update catalog: %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to update catalog, %s", err)
		}

		_, err = c.Backup().Job().WaitForCompletion(ctx, job.ID, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to update catalog, %s", err)
		}

		slaPolicies := []string{}
		for _, policy := range d.Get("backup_sla_policies").(*schema.Set).List() {
			slaPolicies = append(slaPolicies, policy.(string))
		}
		activityId, err = c.Backup().SLAPolicy().AssignVirtualMachine(ctx, &client.BackupAssignVirtualMachineRequest{
			VirtualMachineIds: []string{d.Id()},
			SLAPolicies:       slaPolicies,
		})
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual machine, %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual machine, %s", err)
		}
	}

	return updateVirtualMachine(ctx, d, meta, d.Get("power_state").(string) == "on" && fromScratch)
}

func computeVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	reader := readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
		id := d.Id()
		vm, err := c.Compute().VirtualMachine().Read(ctx, id)
		if err != nil {
			return nil, err
		}
		if vm == nil {
			return nil, nil
		}

		// Normalize the power state so that we can use it as input
		switch vm.PowerState {
		case "running":
			vm.PowerState = "on"
		case "stopped":
			vm.PowerState = "off"
		default:
			return nil, fmt.Errorf("unknown power state %q", vm.PowerState)
		}

		// Normalize the backup_sla_policies
		slaPolicies, err := c.Backup().SLAPolicy().List(ctx, &client.BackupSLAPolicyFilter{
			VirtualMachineId: d.Id(),
		})
		if err != nil {
			return nil, err
		}

		slaPoliciesIds := []string{}
		for _, slaPolicy := range slaPolicies {
			slaPoliciesIds = append(slaPoliciesIds, slaPolicy.ID)
		}

		sw.set("backup_sla_policies", slaPoliciesIds)

		osDisks := []interface{}{}
		for _, osDisk := range d.Get("os_disk").([]interface{}) {
			osDiskId := osDisk.(map[string]interface{})["id"].(string)
			if osDiskId != "" {
				disk, err := c.Compute().VirtualDisk().Read(ctx, osDiskId)
				if err != nil {
					return nil, err
				}
				osDisks = append(osDisks, flattenOSDiskData(disk))
			}
		}

		sw.set("os_disk", osDisks)

		readTags(ctx, sw, c, d.Id())

		return vm, nil
	})

	return reader(ctx, d, meta)
}

func computeVirtualMachineUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return updateVirtualMachine(ctx, d, meta, d.HasChange("power_state"))
}

func updateVirtualMachine(ctx context.Context, d *schema.ResourceData, meta any, updatePower bool) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().VirtualMachine().Update(ctx, &client.UpdateVirtualMachineRequest{
		Id:            d.Id(),
		Ram:           d.Get("memory").(int),
		Cpu:           d.Get("cpu").(int),
		CorePerSocket: d.Get("num_cores_per_socket").(int),
		HotCpuAdd:     d.Get("cpu_hot_add_enabled").(bool),
		HotCpuRemove:  d.Get("cpu_hot_remove_enabled").(bool),
		HotMemAdd:     d.Get("memory_hot_add_enabled").(bool),
		BootOptions: &client.BootOptions{
			BootDelay:        0,
			BootRetryDelay:   10000,
			BootRetryEnabled: false,
			EnterBIOSSetup:   false,
			Firmware:         "bios",
		},
	})
	if err != nil {
		return diag.Errorf("failed to update virtual machine: %s", err)
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("failed to update virtual machine, %s", err)
	}

	if diags := updateTags(ctx, c, d, d.Id(), "vcenter_virtual_machine", "vmware"); diags != nil {
		return diags
	}

	if d.HasChange("name") {
		activityId, err := c.Compute().VirtualMachine().Rename(ctx, d.Id(), d.Get("name").(string))
		if err != nil {
			return diag.Errorf("failed to rename virtual machine, %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to rename virtual machine, %s", err)
		}
	}

	if d.HasChange("guest_operating_system_moref") {
		activityId, err := c.Compute().VirtualMachine().Guest(ctx, d.Id(), &client.UpdateGuestRequest{
			GuestOperatingSystemMoref: d.Get("guest_operating_system_moref").(string),
		})
		if err != nil {
			return diag.Errorf("failed to update virtual machine guest operating system, %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to update virtual machine guest operating system, %s", err)
		}
	}

	if d.HasChange("datacenter_id") || d.HasChange("host_id") || d.HasChange("host_cluster_id") || d.HasChange("datastore_id") || d.HasChange("datastore_cluster_id") {
		activityId, err := c.Compute().VirtualMachine().Relocate(ctx, &client.RelocateVirtualMachineRequest{
			VirtualMachines:    []string{d.Id()},
			Priority:           "highPriority",
			DatacenterId:       d.Get("datacenter_id").(string),
			HostId:             d.Get("host_id").(string),
			HostClusterId:      d.Get("host_cluster_id").(string),
			DatastoreId:        d.Get("datastore_id").(string),
			DatastoreClusterId: d.Get("datastore_cluster_id").(string),
		})
		if err != nil {
			return diag.Errorf("failed to relocate virtual machine, %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to relocate virtual machine, %s", err)
		}
	}

	if d.HasChange("backup_sla_policies") {
		slaPolicies := []string{}
		for _, policy := range d.Get("backup_sla_policies").(*schema.Set).List() {
			slaPolicies = append(slaPolicies, policy.(string))
		}
		activityId, err = c.Backup().SLAPolicy().AssignVirtualMachine(ctx, &client.BackupAssignVirtualMachineRequest{
			VirtualMachineIds: []string{d.Id()},
			SLAPolicies:       slaPolicies,
		})
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual machine, %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual machine, %s", err)
		}
	}

	if d.HasChange("os_disk") {
		for i, osDisk := range d.Get("os_disk").([]interface{}) {
			disk := osDisk.(map[string]interface{})
			if disk["id"].(string) != "" && d.HasChange(fmt.Sprintf("os_disk.%d", i)) {
				activityId, err := c.Compute().VirtualDisk().Update(ctx, &client.UpdateVirtualDiskRequest{
					ID:          disk["id"].(string),
					NewCapacity: disk["capacity"].(int),
					DiskMode:    disk["disk_mode"].(string),
				})
				if err != nil {
					return diag.Errorf("failed to update virtual disk: %s", err)
				}
				_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
				if err != nil {
					return diag.Errorf("failed to update virtual disk, %s", err)
				}
			}
		}
	}

	if updatePower {
		powerState := d.Get("power_state").(string)

		vm, err := c.Compute().VirtualMachine().Read(ctx, d.Id())
		if err != nil {
			return diag.Errorf("failed to read virtual effect: %s", err)
		}

		var recommendations []*client.VirtualMachinePowerRecommendation
		if powerState == "on" {
			recommendations, err = c.Compute().VirtualMachine().Recommendation(ctx, &client.VirtualMachineRecommendationFilter{
				Id:            d.Id(),
				DatacenterId:  vm.DatacenterId,
				HostClusterId: vm.HostClusterId,
			})
			if err != nil {
				return diag.Errorf("failed to find power recommendations: %s", err)
			}
		}

		var recommendation *client.VirtualMachinePowerRecommendation
		if len(recommendations) > 0 {
			recommendation = &client.VirtualMachinePowerRecommendation{
				Key:           recommendations[0].Key,
				HostClusterId: recommendations[0].HostClusterId,
			}
		} else {
			recommendation = nil
		}

		activityId, err = c.Compute().VirtualMachine().Power(ctx, &client.PowerRequest{
			ID:             d.Id(),
			DatacenterId:   vm.DatacenterId,
			PowerAction:    powerState,
			Recommendation: recommendation,
		})
		if err != nil {
			return diag.Errorf("failed to power %s virtual machine: %s", powerState, err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to power %s virtual machine, %s", powerState, err)
		}
	}

	return computeVirtualMachineRead(ctx, d, meta)
}

func computeVirtualMachineDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	vm, err := c.Compute().VirtualMachine().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read virtual effect: %s", err)
	}

	if vm.PowerState == "running" {
		activityId, err := c.Compute().VirtualMachine().Power(ctx, &client.PowerRequest{
			ID:           d.Id(),
			DatacenterId: vm.DatacenterId,
			PowerAction:  "off",
		})
		if err != nil {
			return diag.Errorf("failed to power off virtual machine: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to power off virtual machine, %s", err)
		}
	}

	activityId, err := c.Compute().VirtualMachine().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete virtual machine: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete virtual machine, %s", err)
	}
	return nil
}

func flattenOSDisksData(osDisks []*client.VirtualDisk) []interface{} {
	if osDisks != nil {
		disks := make([]interface{}, len(osDisks))

		for i, osDisk := range osDisks {
			disks[i] = flattenOSDiskData(osDisk)
		}

		return disks
	}

	return make([]interface{}, 0)
}

func flattenOSDiskData(osDisk *client.VirtualDisk) interface{} {
	disk := make(map[string]interface{})

	disk["id"] = osDisk.ID
	disk["machine_manager_id"] = osDisk.MachineManagerId
	disk["name"] = osDisk.Name
	disk["capacity"] = osDisk.Capacity
	disk["disk_unit_number"] = osDisk.DiskUnitNumber
	disk["controller_bus_number"] = osDisk.ControllerBusNumber
	disk["datastore_id"] = osDisk.DatastoreId
	disk["datastore_name"] = osDisk.DatastoreName
	disk["instant_access"] = osDisk.InstantAccess
	disk["native_id"] = osDisk.NativeId
	disk["disk_path"] = osDisk.DiskPath
	disk["provisioning_type"] = osDisk.ProvisioningType
	disk["disk_mode"] = osDisk.DiskMode
	disk["editable"] = osDisk.Editable

	return disk
}
