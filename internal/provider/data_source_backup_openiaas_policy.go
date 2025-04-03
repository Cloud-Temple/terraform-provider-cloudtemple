package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasBackupPolicy() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific backup policy from an Open IaaS infrastructure.",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			id := d.Get("id").(string)
			if id != "" {
				policy, err := c.Backup().OpenIaaS().Policy().Read(ctx, id)
				if err == nil && policy == nil {
					return nil, fmt.Errorf("failed to find backup policy with id %q", id)
				}
				return policy, err
			}

			name := d.Get("name").(string)
			if name != "" {
				policies, err := c.Backup().OpenIaaS().Policy().List(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to list backup policies: %s", err)
				}
				for _, policy := range policies {
					if policy.Name == name {
						return policy, nil
					}
				}
				return nil, fmt.Errorf("failed to find backup policy named %q", name)
			}

			return nil, fmt.Errorf("either id or name must be specified")
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
				Computed:      true,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name"},
			},
			"machine_manager_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Out
			"internal_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"running": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"mode": {
				Type:     schema.TypeString,
				Computed: true,
			},
			// "machine_manager": {
			// 	Type:     schema.TypeList,
			// 	Computed: true,
			// 	Elem: &schema.Resource{
			// 		Schema: map[string]*schema.Schema{
			// 			"id": {
			// 				Type:     schema.TypeString,
			// 				Computed: true,
			// 			},
			// 			"name": {
			// 				Type:     schema.TypeString,
			// 				Computed: true,
			// 			},
			// 		},
			// 	},
			// },
			"schedulers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"temporarily_disabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"retention": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"cron": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"timezone": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}
