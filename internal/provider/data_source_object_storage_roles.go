package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceObjectStorageRoles() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves all available object storage roles.",

		ReadContext: dataSourceObjectStorageRolesRead,

		Schema: map[string]*schema.Schema{
			// Out
			"roles": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of available roles.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the role.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the role.",
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
				},
			},
		},
	}
}

func dataSourceObjectStorageRolesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	roles, err := c.ObjectStorage().Role().List(ctx)
	if err != nil {
		return diag.Errorf("failed to list object storage roles: %s", err)
	}

	rolesList := make([]interface{}, len(roles))
	for i, role := range roles {
		rolesList[i] = helpers.FlattenObjectStorageRole(role)
	}

	if err := d.Set("roles", rolesList); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("object_storage_roles")
	return nil
}
