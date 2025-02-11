package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific virtual machine from an Open IaaS infrastructure.",

		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			c := getClient(meta)
			var virtualMachine *client.OpenIaaSVirtualMachine

			name := d.Get("name").(string)
			if name != "" {
				virtualMachines, err := c.Compute().OpenIaaS().VirtualMachine().List(ctx, &client.OpenIaaSVirtualMachineFilter{
					MachineManagerID: d.Get("machine_manager_id").(string),
				})
				if err != nil {
					return diag.Errorf("failed to find virtual machine named %q: %s", name, err)
				}
				for _, currVirtualMachine := range virtualMachines {
					if currVirtualMachine.Name == name {
						virtualMachine = currVirtualMachine
					}
				}
				if virtualMachine == nil {
					return diag.Errorf("failed to find virtual machine named %q", name)
				}
			} else {
				id := d.Get("id").(string)
				var err error
				virtualMachine, err = c.Compute().OpenIaaS().VirtualMachine().Read(ctx, id)
				if err == nil && virtualMachine == nil {
					return diag.Errorf("failed to find virtual machine with id %q", id)
				}
			}

			sw := newStateWriter(d)

			d.SetId(virtualMachine.ID)
			d.Set("name", virtualMachine.Name)
			d.Set("machine_manager_id", virtualMachine.MachineManager.ID)
			d.Set("internal_id", virtualMachine.InternalID)
			d.Set("power_state", virtualMachine.PowerState)
			d.Set("secure_boot", virtualMachine.SecureBoot)
			d.Set("auto_power_on", virtualMachine.AutoPowerOn)
			d.Set("dvd_drive", []interface{}{
				map[string]interface{}{
					"name":     virtualMachine.DvdDrive.Name,
					"attached": virtualMachine.DvdDrive.Attached,
				},
			})
			d.Set("tools", []interface{}{
				map[string]interface{}{
					"detected": virtualMachine.Tools.Detected,
					"version":  virtualMachine.Tools.Version,
				},
			})
			d.Set("boot_order", virtualMachine.BootOrder)
			d.Set("operating_system_name", virtualMachine.OperatingSystemName)
			d.Set("cpu", virtualMachine.CPU)
			d.Set("num_cores_per_socket", virtualMachine.NumCoresPerSocket)
			d.Set("memory", virtualMachine.Memory)
			d.Set("addresses", []interface{}{
				map[string]interface{}{
					"ipv6": virtualMachine.Addresses.IPv6,
					"ipv4": virtualMachine.Addresses.IPv4,
				},
			})
			d.Set("pool", flattenBaseObject(virtualMachine.Pool))
			d.Set("host", flattenBaseObject(virtualMachine.Host))

			return sw.diags
		},

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},

			// Out
			"internal_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"power_state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"secure_boot": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"auto_power_on": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"dvd_drive": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"attached": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"tools": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"detected": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"version": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"boot_order": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"operating_system_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cpu": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"num_cores_per_socket": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"memory": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ipv6": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ipv4": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"pool": {
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
					},
				},
			},
			"host": {
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
					},
				},
			},
		},
	}
}
