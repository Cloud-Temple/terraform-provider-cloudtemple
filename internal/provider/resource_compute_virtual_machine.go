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

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"virtual_datacenter_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
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
				Type:     schema.TypeInt,
				Optional: true,
				Default:  33554432,
				ForceNew: true,
			},
			"cpu": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  1,
			},
			"guest_operating_system_moref": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"power_state": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "off",
				ValidateFunc: validation.StringInSlice([]string{"on", "off"}, false),
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
			"num_cores_per_socket": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"operating_system_name": {
				Type:     schema.TypeString,
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

func computeVirtualMachineCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	name := d.Get("name").(string)

	activityId, err := c.Compute().VirtualMachine().Create(ctx, &client.CreateVirtualMachineRequest{
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
		return diag.FromErr(err)
	}

	activity, err := c.Activity().WaitForCompletion(ctx, activityId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(activity.ConcernedItems[0].ID)

	if d.Get("power_state").(string) == "on" {
		return computeVirtualMachineUpdate(ctx, d, meta)
	}

	return computeVirtualMachineRead(ctx, d, meta)
}

func computeVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	reader := readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error) {
		id := d.Id()
		if id == "" {
			panic("")
		}
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

		return vm, err
	})

	return reader(ctx, d, meta)
}

func computeVirtualMachineUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	powerState := d.Get("power_state").(string)

	vm, err := c.Compute().VirtualMachine().Read(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	activityId, err := c.Compute().VirtualMachine().Update(ctx, &client.UpdateVirtualMachineRequest{
		Id: d.Id(),
		BootOptions: &client.BootOptions{
			BootDelay:        0,
			BootRetryDelay:   10000,
			BootRetryEnabled: false,
			EnterBIOSSetup:   false,
			Firmware:         "bios",
		},
	})
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId)
	if err != nil {
		return diag.FromErr(err)
	}

	activityId, err = c.Compute().VirtualMachine().Power(ctx, &client.PowerRequest{
		ID:           d.Id(),
		DatacenterId: vm.VirtualDatacenterId,
		PowerAction:  powerState,
	})
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId)
	if err != nil {
		return diag.FromErr(err)
	}

	return computeVirtualMachineRead(ctx, d, meta)
}

func computeVirtualMachineDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().VirtualMachine().Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId)

	return diag.FromErr(err)
}
