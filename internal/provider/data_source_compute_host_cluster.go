package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceHostCluster() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				hostClusters, err := c.Compute().HostCluster().List(ctx, &client.HostClusterFilter{
					Name:             name,
					MachineManagerId: d.Get("machine_manager_id").(string),
					DatacenterId:     d.Get("datacenter_id").(string),
					DatastoreId:      d.Get("datastore_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find host cluster named %q: %s", name, err)
				}
				for _, hc := range hostClusters {
					if hc.Name == name {
						return hc, nil
					}
				}
				return nil, fmt.Errorf("failed to find host cluster named %q", name)
			}

			id := d.Get("id").(string)
			cluster, err := c.Compute().HostCluster().Read(ctx, id)
			if err == nil && cluster == nil {
				return nil, fmt.Errorf("failed to find host cluster with id %q", id)
			}
			return cluster, err
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
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"datastore_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},

			// Out
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hosts": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"metrics": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"total_cpu": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"total_memory": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"total_storage": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"cpu_used": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"memory_used": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"storage_used": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"virtual_machines_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}
