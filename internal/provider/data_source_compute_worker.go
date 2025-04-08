package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceWorker() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				worker, err := client.Compute().Worker().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if worker == nil {
					return nil, fmt.Errorf("failed to find worker with id %q", id)
				}
				return worker, nil
			}

			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				workers, err := client.Compute().Worker().List(ctx, "")
				if err != nil {
					return nil, fmt.Errorf("failed to list workers: %s", err)
				}
				for _, worker := range workers {
					if worker.Name == name {
						return worker, nil
					}
				}
				return nil, fmt.Errorf("failed to find worker with name %q", name)
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
			"full_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vendor": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"build": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"locale_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"locale_build": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"os_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"product_line_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"api_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"api_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"instance_uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"license_product_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"license_product_version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tenant_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
