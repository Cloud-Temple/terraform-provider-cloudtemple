package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualMachines() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			virtualMachines, err := client.Compute().VirtualMachine().List(ctx, true, "", false, false, nil, nil, nil, nil, nil)
			return map[string]interface{}{
				"id":               "virtual_machines",
				"virtual_machines": virtualMachines,
			}, err
		}),

		Schema: map[string]*schema.Schema{
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
				},
			},
		},
	}
}
