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
		Description: "",

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
			"clone_virtual_machine_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the virtual machine to clone. Conflicts with `guest_operating_system_moref`.",
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"guest_operating_system_moref"},
				AtLeastOneOf:  []string{"clone_virtual_machine_id", "guest_operating_system_moref"},
				ValidateFunc:  validation.IsUUID,
			},
			"guest_operating_system_moref": {
				Type:          schema.TypeString,
				Description:   "The operating system to launch the virtual machine with. Conflicts with `clone_virtual_machine_id`.",
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"clone_virtual_machine_id"},
				AtLeastOneOf:  []string{"clone_virtual_machine_id", "guest_operating_system_moref"},
			},
			"virtual_datacenter_id": {
				Type:         schema.TypeString,
				Description:  "The datacenter to start the virtual machine in.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_id": {
				Type:         schema.TypeString,
				Description:  "The host to start the virtual machine on.",
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
				Description:  "The host cluster to start the virtual machine on.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"datastore_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"datastore_cluster_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
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
	cloneVirtualMachineId := d.Get("clone_virtual_machine_id").(string)
	if cloneVirtualMachineId == "" {
		activityId, err = c.Compute().VirtualMachine().Create(ctx, &client.CreateVirtualMachineRequest{
			Name:                      name,
			DatacenterId:              d.Get("virtual_datacenter_id").(string),
			HostId:                    d.Get("host_id").(string),
			HostClusterId:             d.Get("host_cluster_id").(string),
			DatastoreId:               d.Get("datastore_id").(string),
			DatastoreClusterId:        d.Get("datastore_cluster_id").(string),
			Memory:                    d.Get("memory").(int),
			CPU:                       d.Get("cpu").(int),
			GuestOperatingSystemMoref: d.Get("guest_operating_system_moref").(string),
		})
		if err != nil {
			return diag.Errorf("failed to create virtual machine: %s", err)
		}

		activity, err := c.Activity().WaitForCompletion(ctx, activityId)
		if err != nil {
			return diag.Errorf("failed to create virtual machine, %s", err)
		}

		d.SetId(activity.ConcernedItems[0].ID)
	} else {
		activityId, err = c.Compute().VirtualMachine().Clone(ctx, &client.CloneVirtualMachineRequest{
			Name:              name,
			VirtualMachineId:  cloneVirtualMachineId,
			PowerOn:           d.Get("power_state").(string) == "on",
			DatacenterId:      d.Get("virtual_datacenter_id").(string),
			HostClusterId:     d.Get("host_cluster_id").(string),
			HostId:            d.Get("host_id").(string),
			DatatoreClusterId: d.Get("datastore_cluster_id").(string),
			DatastoreId:       d.Get("datastore_id").(string),
		})
		if err != nil {
			return diag.Errorf("failed to clone virtual machine: %s", err)
		}

		activity, err := c.Activity().WaitForCompletion(ctx, activityId)
		if err != nil {
			return diag.Errorf("failed to clone virtual machine, %s", err)
		}

		d.SetId(activity.State["completed"].Result)
	}

	return updateVirtualMachine(ctx, d, meta, d.Get("power_state").(string) == "on" && cloneVirtualMachineId == "")
}

func computeVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	reader := readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
		id := d.Id()
		vm, err := client.Compute().VirtualMachine().Read(ctx, id)
		if err == nil && vm == nil {
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

		readTags(ctx, sw, client, d.Id())

		return vm, err
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
	_, err = c.Activity().WaitForCompletion(ctx, activityId)
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
		_, err = c.Activity().WaitForCompletion(ctx, activityId)
		if err != nil {
			return diag.Errorf("failed to rename virtual machine, %s", err)
		}
	}

	if updatePower {
		powerState := d.Get("power_state").(string)

		vm, err := c.Compute().VirtualMachine().Read(ctx, d.Id())
		if err != nil {
			return diag.Errorf("failed to read virtual effect: %s", err)
		}

		activityId, err = c.Compute().VirtualMachine().Power(ctx, &client.PowerRequest{
			ID:           d.Id(),
			DatacenterId: vm.VirtualDatacenterId,
			PowerAction:  powerState,
		})
		if err != nil {
			return diag.Errorf("failed to power %s virtual machine: %s", powerState, err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId)
		if err != nil {
			return diag.Errorf("failed to power %s virtual machine, %s", powerState, err)
		}
	}

	return computeVirtualMachineRead(ctx, d, meta)
}

func computeVirtualMachineDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().VirtualMachine().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete virtual machine: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId); err != nil {
		return diag.Errorf("failed to delete virtual machine, %s", err)
	}
	return nil
}
