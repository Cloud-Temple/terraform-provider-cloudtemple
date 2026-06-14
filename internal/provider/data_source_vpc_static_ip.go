package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVPCStaticIP() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific VPC static IP, by ID or by MAC address.",

		ReadContext: dataSourceVPCStaticIPRead,

		Schema: map[string]*schema.Schema{
			// In - exactly one of id or mac_address
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IsUUID,
				ExactlyOneOf: []string{"id", "mac_address"},
				Description:  "The ID of the static IP to retrieve. Exactly one of `id` or `mac_address` must be set.",
			},
			"mac_address": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"id", "mac_address"},
				Description:  "The MAC address of the static IP to retrieve. Exactly one of `id` or `mac_address` must be set.",
			},

			// Out
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The static IP address.",
			},
			"source": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The source of the static IP (one of `xoa`, `vmware`, `custom`).",
			},
			"resource_description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the resource, for a custom source.",
			},
			"virtual_machine_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the virtual machine associated with this static IP, if any.",
			},
			"network_adapter_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the network adapter associated with this static IP, if any.",
			},
			"floating_ip_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the floating IP bound to this static IP, if any.",
			},
			"floating_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The address of the floating IP bound to this static IP, if any.",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the VPC this static IP belongs to.",
			},
			"private_network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the private network this static IP belongs to.",
			},
		},
	}
}

func dataSourceVPCStaticIPRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	id := d.Get("id").(string)
	mac := d.Get("mac_address").(string)

	var (
		staticIP *client.StaticIP
		err      error
	)
	switch {
	case id != "":
		staticIP, err = c.VPC().StaticIP().Read(ctx, id)
	case mac != "":
		staticIP, err = c.VPC().StaticIP().ReadByMAC(ctx, mac)
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if staticIP == nil {
		if id != "" {
			return diag.FromErr(fmt.Errorf("failed to find static IP with id %q", id))
		}
		return diag.FromErr(fmt.Errorf("failed to find static IP with MAC address %q", mac))
	}

	d.SetId(staticIP.ID)

	for k, v := range helpers.FlattenStaticIP(staticIP) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
