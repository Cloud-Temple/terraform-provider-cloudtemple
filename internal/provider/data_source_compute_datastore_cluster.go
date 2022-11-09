package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDatastoreCluster() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				clusters, err := client.Compute().DatastoreCluster().List(ctx, "", "", "", "")
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
			cluster, err := client.Compute().DatastoreCluster().Read(ctx, id)
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
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},

			// Out
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_id": {
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
