package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasVirtualDisk() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific virtual disk from an Open IaaS infrastructure.",

		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			c := getClient(meta)
			var disk *client.OpenIaaSVirtualDisk

			name := d.Get("name").(string)
			if name != "" {
				disks, err := c.Compute().OpenIaaS().VirtualDisk().List(ctx, d.Get("virtual_machine_id").(string))
				if err != nil {
					return diag.Errorf("failed to find virtual disk named %q: %s", name, err)
				}
				for _, currDisk := range disks {
					if currDisk.Name == name {
						disk = currDisk
					}
				}
				return diag.Errorf("failed to find virtual disk named %q", name)
			} else {
				id := d.Get("id").(string)
				var err error
				disk, err = c.Compute().OpenIaaS().VirtualDisk().Read(ctx, id)
				if err == nil && disk == nil {
					return diag.Errorf("failed to find virtual disk with id %q", id)
				}
			}

			sw := newStateWriter(d)

			d.SetId(disk.ID)
			d.Set("name", disk.Name)
			d.Set("description", disk.Description)
			d.Set("size", disk.Size)
			d.Set("usage", disk.Usage)
			d.Set("snapshots", disk.Snapshots)
			d.Set("storage_repository", []interface{}{
				map[string]interface{}{
					"id":          disk.StorageRepository.ID,
					"name":        disk.StorageRepository.Name,
					"description": disk.StorageRepository.Description,
				},
			})
			var virtualMachines []interface{}
			for _, vm := range disk.VirtualMachines {
				virtualMachines = append(virtualMachines, map[string]interface{}{
					"id":        vm.ID,
					"read_only": vm.ReadOnly,
				})
			}
			d.Set("virtual_machines", virtualMachines)

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
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},

			// Out
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"usage": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"snapshots": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"storage_repository": {
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
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"virtual_machines": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"read_only": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}
