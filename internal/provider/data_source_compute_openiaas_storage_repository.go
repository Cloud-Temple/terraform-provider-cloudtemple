package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasStorageRepository() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific storage repository from an Open IaaS infrastructure.",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			id := d.Get("id").(string)
			if id != "" {
				repository, err := c.Compute().OpenIaaS().StorageRepository().Read(ctx, id)
				if err != nil {
					return nil, fmt.Errorf("failed to find storage repository with id %q: %s", id, err)
				}
				return repository, nil
			}

			name := d.Get("name").(string)
			if name != "" {
				types := d.Get("types").([]interface{})
				stringTypes := make([]string, 0, len(types))
				for _, v := range types {
					stringTypes = append(stringTypes, v.(string))
				}
				repositories, err := c.Compute().OpenIaaS().StorageRepository().List(ctx, &client.StorageRepositoryFilter{
					MachineManagerId: d.Get("machine_manager_id").(string),
					Types:            stringTypes,
					Shared:           d.Get("shared").(bool),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find storage repository named %q: %s", name, err)
				}
				for _, repository := range repositories {
					if repository.Name == name {
						return repository, nil
					}
				}
				return nil, fmt.Errorf("failed to find storage repository named %q", name)
			}

			return nil, fmt.Errorf("id and name are required to find a storage repository")
		}),

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
			"types": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
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
			"machine_manager": {
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
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
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
