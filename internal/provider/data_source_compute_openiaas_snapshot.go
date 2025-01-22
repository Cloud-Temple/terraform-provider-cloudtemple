package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific snapshot from an Open IaaS infrastructure.",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			id := d.Get("id").(string)
			var snapshot *client.OpenIaaSSnapshot
			var err error
			if id != "" {
				snapshot, err = c.Compute().OpenIaaS().Snapshot().Read(ctx, id)
				if err == nil && snapshot == nil {
					return nil, fmt.Errorf("failed to find snapshot with id %q", id)
				}
				return snapshot, err
			}

			name := d.Get("name").(string)
			if name != "" {
				snapshots, err := c.Compute().OpenIaaS().Snapshot().List(ctx, d.Get("virtual_machine_id").(string))
				if err != nil {
					return nil, fmt.Errorf("failed to list snapshots: %s", err)
				}
				for _, snapshot := range snapshots {
					if snapshot.Name == name {
						return snapshot, nil
					}
				}
				return nil, fmt.Errorf("failed to find snapshot named %q", name)
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
			"create_time": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}
