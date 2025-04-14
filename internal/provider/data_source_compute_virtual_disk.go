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
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"virtual_machine_id"},
			},
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				RequiredWith:  []string{"name"},
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
			},

			// Out
			"machine_manager_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"disk_unit_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"controller_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"controller_type": {
				Type:     schema.TypeString,
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
			"disk_mode": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"editable": {
				Type:     schema.TypeBool,
				Computed: true,
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
