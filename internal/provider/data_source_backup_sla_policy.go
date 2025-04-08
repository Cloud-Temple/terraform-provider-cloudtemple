package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceBackupSLAPolicy() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				policies, err := client.Backup().SLAPolicy().List(ctx, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to find SLA policy named %q: %s", name, err)
				}
				for _, policy := range policies {
					if policy.Name == name {
						return policy, nil
					}
				}
				return nil, fmt.Errorf("failed to find SLA policy named %q", name)
			}

			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				policy, err := client.Backup().SLAPolicy().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if policy == nil {
					return nil, fmt.Errorf("failed to find SLA policy with id %q", id)
				}
				return policy, nil
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

			// Out
			"sub_policies": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"use_encryption": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"software": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"site": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"retention": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"age": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"trigger": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"frequency": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"type": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"activate_date": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"target": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"href": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"resource_type": {
										Type:     schema.TypeString,
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
