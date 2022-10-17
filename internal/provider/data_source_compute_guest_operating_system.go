package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceGuestOperatingSystem() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			gos, err := client.Compute().GuestOperatingSystem().Read(ctx, d.Get("machine_manager_id").(string), d.Get("moref").(string))
			if gos != nil {
				d.SetId(gos.Moref)
			}
			if err == nil && gos == nil {
				return nil, fmt.Errorf("failed to find guest operating system")
			}
			return gos, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"moref": {
				Type:     schema.TypeString,
				Required: true,
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},

			// Out
			"family": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"full_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
