package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func readResource(read func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, []string, error)) func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
		client := getClient(meta)
		res, skip, err := read(ctx, client, d)
		if err != nil {
			return diag.FromErr(err)
		}
		if res == nil {
			d.SetId("")
			return nil
		}

		sw := newStateWriter(d)
		sw.save(res, skip)

		return sw.diags
	}
}

func readFullResource(read func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error)) func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return readResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, []string, error) {
		resource, err := read(ctx, client, d)
		return resource, nil, err
	})
}
