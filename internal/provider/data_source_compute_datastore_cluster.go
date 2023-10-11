package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceDatastoreCluster() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				clusters, err := c.Compute().DatastoreCluster().List(ctx, &client.DatastoreClusterFilter{
					Name:             name,
					MachineManagerId: d.Get("machine_manager_id").(string),
					DatacenterId:     d.Get("datacenter_id").(string),
					HostId:           d.Get("host_id").(string),
					HostClusterId:    d.Get("host_cluster_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find datastore cluster named %q: %s", name, err)
				}
				for _, c := range clusters {
					if c.Name == name {
						return c, nil
					}
				}
				return nil, fmt.Errorf("failed to find datastore cluster named %q", name)
			}

			id := d.Get("id").(string)
			cluster, err := c.Compute().DatastoreCluster().Read(ctx, id)
			if err == nil && cluster == nil {
				return nil, fmt.Errorf("failed to find datastore cluster with id %q", id)
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
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name", "datacenter_id"},
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
			},
			"host_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
			},

			// Out
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastores": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metrics": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"free_capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"max_capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"default_vm_behavior": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"load_balance_interval": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"space_threshold_mode": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"space_utilization_threshold": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"min_space_utilization_difference": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"reservable_percent_threshold": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"reservable_threshold_mode": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"io_latency_threshold": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"io_load_imbalance_threshold": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"io_load_balance_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}
