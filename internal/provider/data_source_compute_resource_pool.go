package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceResourcePool() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				resourcePools, err := client.Compute().ResourcePool().List(ctx, "", "", "")
				if err != nil {
					return nil, fmt.Errorf("failed to find resource pool named %q: %s", name, err)
				}
				for _, pool := range resourcePools {
					if pool.Name == name {
						return pool, nil
					}
				}
				return nil, fmt.Errorf("failed to find resource pool named %q", name)
			}

			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				pool, err := client.Compute().ResourcePool().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if pool == nil {
					return nil, fmt.Errorf("failed to find resource pool with id %q", id)
				}
				return pool, nil
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

			// Out
			"machine_manager_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"parent": {
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
						"cpu": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"max_usage": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"reservation_used": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"memory": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"max_usage": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"reservation_used": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"ballooned_memory": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
