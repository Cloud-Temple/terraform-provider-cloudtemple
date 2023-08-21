package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func getBy(
	ctx context.Context,
	d *schema.ResourceData,
	typ string,
	getter func(id string) (any, error),
	list func(d *schema.ResourceData) (any, error),
	attrs []string) (interface{}, error) {
	for _, attr := range attrs {
		expected := d.Get(attr).(string)
		if expected != "" {
			items, err := list(d)
			if err != nil {
				return nil, fmt.Errorf("failed to find %s with %s %q: %s", typ, attr, expected, err)
			}
			items_ := reflect.ValueOf(items)
			for i := 0; i < items_.Len(); i++ {
				item := items_.Index(i).Elem()
				for _, field := range reflect.VisibleFields(item.Type()) {
					name := field.Tag.Get("terraform")
					if name == attr && item.FieldByName(field.Name).Interface() == expected {
						res := item.Interface()
						return &res, nil
					}
				}
			}
			return nil, fmt.Errorf("failed to find %s with %s %q", typ, attr, expected)
		}
	}

	id := d.Get("id").(string)
	item, err := getter(id)
	if err != nil {
		return nil, err
	}
	if reflect.ValueOf(item).Kind() == reflect.Ptr && reflect.ValueOf(item).IsNil() {
		return nil, fmt.Errorf("failed to find %s with id %q", typ, id)
	}
	return item, err
}

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
