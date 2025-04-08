package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				virtualMachineId := d.Get("virtual_machine_id").(string)
				if virtualMachineId == "" {
					return nil, fmt.Errorf("virtual_machine_id is required when searching by name")
				}

				adapters, err := client.Compute().NetworkAdapter().List(ctx, virtualMachineId)
				if err != nil {
					return nil, fmt.Errorf("failed to find network adapter named %q: %s", name, err)
				}
				for _, adapter := range adapters {
					if adapter.Name == name {
						return adapter, nil
					}
				}
				return nil, fmt.Errorf("failed to find network adapter named %q", name)
			}

			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				adapter, err := client.Compute().NetworkAdapter().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if adapter == nil {
					return nil, fmt.Errorf("failed to find network adapter with id %q", id)
				}
				return adapter, nil
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
				RequiredWith:  []string{"virtual_machine_id"},
			},
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				RequiredWith:  []string{"name"},
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
			},

			// Out
			"network_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"mac_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"mac_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"connected": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"auto_connect": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}
