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

func dataSourceVirtualDisk() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific virtual disk from a vCenter infrastructure.",

		ReadContext: dataSourceVirtualDiskRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the virtual disk to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"virtual_machine_id"},
				Description:   "The name of the virtual disk to retrieve. Conflicts with `id`. Requires `virtual_machine_id`.",
			},
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				RequiredWith:  []string{"name"},
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the virtual machine that the disk is attached to. Required when using `name`.",
			},

			// Out
			"machine_manager_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the machine manager (vCenter) where this virtual disk is located.",
			},
			"capacity": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The capacity of the virtual disk in Bytes.",
			},
			"disk_unit_number": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The unit number of the disk on its controller.",
			},
			"controller_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the controller this disk is attached to.",
			},
			"controller_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the controller (e.g., SCSI, IDE, NVME).",
			},
			"controller_bus_number": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The bus number of the controller.",
			},
			"datastore_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the datastore where this virtual disk is stored.",
			},
			"datastore_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the datastore where this virtual disk is stored.",
			},
			"instant_access": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the disk is an instant access disk.",
			},
			"native_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The native ID of the disk in the hypervisor.",
			},
			"disk_path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The path to the disk file in the datastore.",
			},
			"provisioning_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The provisioning type of the disk.",
			},
			"disk_mode": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The disk mode.",
			},
			"editable": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the disk is editable.",
			},
		},
	}
}

// dataSourceVirtualDiskRead lit un disque virtuel et le mappe dans le state Terraform
func dataSourceVirtualDiskRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var disk *client.VirtualDisk
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		disks, err := c.Compute().VirtualDisk().List(ctx, &client.VirtualDiskFilter{
			Name:             name,
			VirtualMachineID: d.Get("virtual_machine_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find disk named %q: %s", name, err))
		}
		for _, d := range disks {
			if d.Name == name {
				disk = d
				break
			}
		}
		if disk == nil {
			return diag.FromErr(fmt.Errorf("failed to find disk named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			disk, err = c.Compute().VirtualDisk().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if disk == nil {
				return diag.FromErr(fmt.Errorf("failed to find disk with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(disk.ID)

	// Mapper les données en utilisant la fonction helper
	diskData := helpers.FlattenVirtualDisk(disk)

	// Définir les données dans le state
	for k, v := range diskData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
