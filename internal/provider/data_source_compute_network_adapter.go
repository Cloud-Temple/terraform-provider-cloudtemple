package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error) {
			id := d.Get("id").(string)
			adapter, err := client.Compute().NetworkAdapter().Read(ctx, id)
			if err == nil && adapter == nil {
				return nil, fmt.Errorf("failed to find network adapter with id %q", id)
			}
			return adapter, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},

			// Out
			"virtual_machine_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
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
