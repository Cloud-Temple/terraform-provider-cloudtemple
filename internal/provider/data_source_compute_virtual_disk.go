package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVirtualDisk() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				disks, err := c.Compute().VirtualDisk().List(ctx, d.Get("virtual_machine_id").(string))
				if err != nil {
					return nil, fmt.Errorf("failed to find disk named %q: %s", name, err)
				}
				for _, disk := range disks {
					if disk.Name == name {
						return disk, nil
					}
				}
				return nil, fmt.Errorf("failed to find disk named %q", name)
			}

			id := d.Get("id").(string)
			if id != "" {
				disk, err := c.Compute().VirtualDisk().Read(ctx, id)
				if err != nil || disk == nil {
					return nil, fmt.Errorf("failed to find disk with id %q", id)
				}
				return disk, err
			}

			return nil, fmt.Errorf("either id or name must be specified")
		}),

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
			"machine_manager_name": {
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
