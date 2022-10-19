package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGuestOperatingSystems() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceGuestOperatingSystemsRead,
	}
}

func dataSourceGuestOperatingSystemsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return nil
}
