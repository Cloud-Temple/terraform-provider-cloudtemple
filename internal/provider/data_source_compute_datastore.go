package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceDatastore() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				datastores, err := c.Compute().Datastore().List(ctx, &client.DatastoreFilter{
					Name:             name,
					MachineManagerId: d.Get("machine_manager_id").(string),
					DatacenterId:     d.Get("datacenter_id").(string),
					HostId:           d.Get("host_id").(string),
					HostClusterId:    d.Get("host_cluster_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find datastore named %q: %s", name, err)
				}
				for _, datastore := range datastores {
					if datastore.Name == name {
						return datastore, nil
					}
				}
				return nil, fmt.Errorf("failed to find datastore named %q", name)
			}

			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				datastore, err := c.Compute().Datastore().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if datastore == nil {
					return nil, fmt.Errorf("failed to find datastore with id %q", id)
				}
				return datastore, nil
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
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"host_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
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
			"max_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"free_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"accessible": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"maintenance_status": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"unique_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"virtual_machines_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"hosts_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"hosts_names": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"associated_folder": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
