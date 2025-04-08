package provider

import (
	"context"
	"reflect"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func readResource(read func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (any, []string, error)) func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
		client := getClient(meta)
		sw := newStateWriter(d)

		res, skip, err := read(ctx, client, d, sw)
		if err != nil {
			return diag.FromErr(err)
		}
		if res == nil || (reflect.ValueOf(res).Kind() == reflect.Ptr && reflect.ValueOf(res).IsNil()) {
			d.SetId("")
			return nil
		}

		sw.save(res, skip)

		return sw.diags
	}
}

func readFullResource(read func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error)) func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return readResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, []string, error) {
		resource, err := read(ctx, client, d, sw)
		return resource, nil, err
	})
}

func exists[T any](data []T, f func(T) bool) bool {
	for _, v := range data {
		if f(v) {
			return true
		}
	}

	return false
}

func flattenBaseObject(obj client.BaseObject) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"id":   obj.ID,
			"name": obj.Name,
		},
	}
}
