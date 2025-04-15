package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenIaasVirtualMachines() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all virtual machines from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSVirtualMachinesRead,

		Schema: map[string]*schema.Schema{
			// In
			"machine_manager_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter virtual machines by the ID of the machine manager they belong to. If not specified, returns virtual machines from all machine managers.",
			},

			// Out
			"virtual_machines": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of virtual machines matching the filter criteria.",

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
						"internal_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internal identifier of the virtual machine in the Open IaaS system.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this virtual machine belongs to.",
						},
						"power_state": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The current power state of the virtual machine (e.g., Running, Halted, Paused, ...).",
						},
						"secure_boot": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether secure boot is enabled for the virtual machine.",
						},
						"boot_firmware": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The boot firmware type used by the virtual machine (e.g., BIOS, EFI).",
						},
						"auto_power_on": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the virtual machine is configured to automatically power on when the host starts.",
						},
						"dvd_drive": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the virtual machine's DVD drive.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the DVD drive.",
									},
									"attached": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether the DVD drive is attached to the virtual machine.",
									},
								},
							},
						},
						"tools": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the virtualization tools installed in the virtual machine.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"detected": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether virtualization tools are detected in the virtual machine.",
									},
									"version": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The version of the virtualization tools installed in the virtual machine.",
									},
								},
							},
						},
						"boot_order": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The boot order of the virtual machine, listing devices in the order they will be tried during boot.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"operating_system_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the operating system installed on the virtual machine.",
						},
						"cpu": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of virtual CPUs allocated to the virtual machine.",
						},
						"num_cores_per_socket": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of cores per CPU socket in the virtual machine.",
						},
						"memory": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The amount of memory allocated to the virtual machine in Bytes.",
						},
						"addresses": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The network addresses assigned to the virtual machine.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ipv6": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The IPv6 address assigned to the virtual machine.",
									},
									"ipv4": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The IPv4 address assigned to the virtual machine.",
									},
								},
							},
						},
						"host_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the host the virtual machine is running on.",
						},
						"pool_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the resource pool the virtual machine belongs to.",
						},
					},
				},
			},
		},
	}
}

// computeOpenIaaSVirtualMachinesRead lit les machines virtuelles OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSVirtualMachinesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les machines virtuelles OpenIaaS
	vms, err := c.Compute().OpenIaaS().VirtualMachine().List(ctx, &client.OpenIaaSVirtualMachineFilter{
		MachineManagerID: d.Get("machine_manager_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_virtual_machines")

	// Mapper manuellement les données en utilisant la fonction helper
	tfVMs := make([]map[string]interface{}, len(vms))
	for i, vm := range vms {
		tfVMs[i] = helpers.FlattenOpenIaaSVirtualMachine(vm)
		tfVMs[i]["id"] = vm.ID
	}

	// Définir les données dans le state
	if err := d.Set("virtual_machines", tfVMs); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
