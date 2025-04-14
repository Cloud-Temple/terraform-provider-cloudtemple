package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific virtual machine from a vCenter infrastructure.",

		ReadContext: dataSourceVirtualMachineRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name"},
			},

			// Out
			"machine_manager_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastore_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"host_cluster_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastore_cluster_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastore_id": {
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
			"expose_hardware_virtualization": {
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
			"datacenter_id": {
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
	}
}

// dataSourceVirtualMachineRead lit une machine virtuelle et la mappe dans le state Terraform
func dataSourceVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var vm *client.VirtualMachine
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		virtualMachines, err := c.Compute().VirtualMachine().List(ctx, &client.VirtualMachineFilter{
			Name:             name,
			MachineManagerID: d.Get("machine_manager_id").(string),
			AllOptions:       true,
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual machine named %q: %s", name, err))
		}
		for _, v := range virtualMachines {
			if v.Name == name {
				vm = v
				break
			}
		}
		if vm == nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual machine named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			vm, err = c.Compute().VirtualMachine().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if vm == nil {
				return diag.FromErr(fmt.Errorf("failed to find virtual machine with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(vm.ID)

	// Mapper les données en utilisant la fonction helper
	vmData := helpers.FlattenVirtualMachine(vm)

	// Définir les données dans le state
	for k, v := range vmData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
