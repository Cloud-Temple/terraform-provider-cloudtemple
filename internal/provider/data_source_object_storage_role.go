package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceObjectStorageRole() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves a specific object storage role by name.",

		ReadContext: dataSourceObjectStorageRoleRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the role to retrieve.",
			},

			// Out
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the role.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the role.",
			},
			"permissions": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of permissions granted by this role.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceObjectStorageRoleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	name := d.Get("name").(string)

	roles, err := c.ObjectStorage().Role().List(ctx)
	if err != nil {
		return diag.Errorf("failed to list object storage roles: %s", err)
	}

	var found *client.ObjectStorageRole
	for _, role := range roles {
		if role.Name == name {
			found = role
			break
		}
	}

	if found == nil {
		return diag.Errorf("object storage role with name %q not found", name)
	}

	roleData := helpers.FlattenObjectStorageRole(found)
	for k, v := range roleData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(found.Name)
	return nil
}
