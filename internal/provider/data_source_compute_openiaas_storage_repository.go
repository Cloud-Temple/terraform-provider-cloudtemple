package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasStorageRepository() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific storage repository from an Open IaaS infrastructure.",

		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			c := getClient(meta)
			var sr *client.OpenIaaSStorageRepository

			name := d.Get("name").(string)
			if name != "" {
				stringTypes := make([]string, 0, 1)
				stringTypes = append(stringTypes, d.Get("type").(string))
				repositories, err := c.Compute().OpenIaaS().StorageRepository().List(ctx, &client.StorageRepositoryFilter{
					MachineManagerId: d.Get("machine_manager_id").(string),
					StorageTypes:     stringTypes,
					Shared:           d.Get("shared").(bool),
				})
				if err != nil {
					return diag.Errorf("failed to find storage repository named %q: %s", name, err)
				}
				for _, currSr := range repositories {
					if currSr.Name == name {
						sr = currSr
					}
				}
				return diag.Errorf("failed to find storage repository named %q", name)
			} else {
				id := d.Get("id").(string)
				var err error
				sr, err = c.Compute().OpenIaaS().StorageRepository().Read(ctx, id)
				if err != nil && sr == nil {
					return diag.Errorf("failed to find storage repository with id %q: %s", id, err)
				}
			}

			sw := newStateWriter(d)

			d.SetId(sr.ID)
			d.Set("name", sr.Name)
			d.Set("machine_manager_id", sr.MachineManager.ID)
			d.Set("internal_id", sr.InternalId)
			d.Set("description", sr.Description)
			d.Set("maintenance_status", sr.MaintenanceStatus)
			d.Set("accessible", sr.Accessible)
			d.Set("storage_type", sr.StorageType)
			d.Set("shared", sr.Shared)
			d.Set("free_capacity", sr.FreeCapacity)
			d.Set("max_capacity", sr.MaxCapacity)
			d.Set("virtual_disks", sr.VirtualDisks)
			d.Set("pool", []interface{}{
				map[string]interface{}{
					"id":   sr.Pool.ID,
					"name": sr.Pool.Name,
				},
			})
			d.Set("host", []interface{}{
				map[string]interface{}{
					"id":   sr.Host.ID,
					"name": sr.Host.Name,
				},
			})

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
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Available values are: ext, lvm, lvmoiscsi, lvmohba, nfs, smb, iso, nfs_iso, cifs",
				ValidateFunc: validation.StringInSlice([]string{
					"ext",
					"lvm",
					"lvmoiscsi",
					"lvmohba",
					"nfs",
					"smb",
					"iso",
					"nfs_iso",
					"cifs",
				}, true),
			},
			"shared": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			// Out
			"internal_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"maintenance_status": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"accessible": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			// "type": {
			// 	Type:     schema.TypeString,
			// 	Computed: true,
			// },
			"free_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"max_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"virtual_disks": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
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
