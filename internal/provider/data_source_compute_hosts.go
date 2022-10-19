package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHosts() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceHostsRead,
	}
}

func dataSourceHostsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return nil
}
