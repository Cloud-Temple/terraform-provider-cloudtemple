package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVcenters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceVcentersRead,
	}
}

func dataSourceVcentersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return nil
}
