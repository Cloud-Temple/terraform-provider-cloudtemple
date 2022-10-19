package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualMachines() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceVirtualMachinesRead,
	}
}

func dataSourceVirtualMachinesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return nil
}
