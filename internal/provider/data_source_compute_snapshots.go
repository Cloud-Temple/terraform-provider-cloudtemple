package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSnapshots() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceSnapshotsRead,
	}
}

func dataSourceSnapshotsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return nil
}
