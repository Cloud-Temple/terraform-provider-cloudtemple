package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func readTags(ctx context.Context, sw *stateWriter, client *client.Client, id string) {
	tags, err := client.Tag().Resource().Read(ctx, id)
	if err != nil {
		sw.diags = append(sw.diags, diag.Errorf("failed to read resource %q tags: %s", id, err)...)
	}

	mTags := map[string]interface{}{}
	for _, tag := range tags {
		mTags[tag.Key] = tag.Value
	}
	sw.set("tags", mTags)
}

func updateTags(ctx context.Context, c *client.Client, d *schema.ResourceData, id, typ, source string) diag.Diagnostics {
	if !d.HasChange("tags") {
		return nil
	}

	wanted := d.Get("tags").(map[string]interface{})
	existing, err := c.Tag().Resource().Read(ctx, id)
	if err != nil {
		return diag.Errorf("failed to read tags: %s", err)
	}

	for _, tag := range existing {
		// The tag exists so we check if it has the correct value and if so
		// we skip updating it
		if value, found := wanted[tag.Key]; found && tag.Value == value.(string) {
			delete(wanted, tag.Key)
			continue
		}

		// If the tag should not exist or if it exists but with the wrong value
		// we remove it
		err := c.Tag().Resource().Delete(ctx, id, tag.Key)
		if err != nil {
			return diag.Errorf("failed to delete tag: %s", err)
		}
	}

	for key, value := range wanted {
		err := c.Tag().Resource().Create(ctx, &client.CreateTagRequest{
			Key:   key,
			Value: value.(string),
			Resources: []*client.CreateTagRequestResource{
				{
					UUID:   id,
					Type:   typ,
					Source: source,
				},
			},
		})
		if err != nil {
			return diag.Errorf("failed to create tag: %s", err)
		}
	}
	return nil
}
