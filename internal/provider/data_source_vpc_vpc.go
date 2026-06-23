package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVPCVPC() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific VPC.",

		ReadContext: dataSourceVPCVPCRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the VPC to retrieve.",
			},

			// Out
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the VPC.",
			},
			"internet_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internet IP address assigned to the VPC, if any.",
			},
			"private_network_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of private networks in this VPC.",
			},
			"static_ip_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of static IPs in this VPC.",
			},
			"floating_ip_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of floating IPs in this VPC.",
			},
		},
	}
}

func dataSourceVPCVPCRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	id := d.Get("id").(string)

	vpc, err := c.VPC().VPC().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if vpc == nil {
		return diag.FromErr(fmt.Errorf("failed to find VPC with id %q", id))
	}

	d.SetId(vpc.ID)

	for k, v := range helpers.FlattenVPC(vpc) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
