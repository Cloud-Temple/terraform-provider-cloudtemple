package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupJob() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				jobs, err := client.Backup().Job().List(ctx, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to find job named %q: %s", name, err)
				}
				for _, job := range jobs {
					if job.Name == name {
						return job, nil
					}
				}
				return nil, fmt.Errorf("failed to find job named %q", name)
			}

			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				job, err := client.Backup().Job().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if job == nil {
					return nil, fmt.Errorf("failed to find job with id %q", id)
				}
				return job, nil
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
				ValidateFunc:  IsNumber,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},

			// Out
			"display_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"policy_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
