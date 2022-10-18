package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceUserRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Description: "",
				Type:        schema.TypeString,
				Required:    true,
			},

			// Out
			"internal_id": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"type": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"source": {
				Description: "",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"source_id": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"email_verified": {
				Description: "",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"email": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceUserRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	userID := d.Get("id").(string)
	user, err := client.IAM().User().Read(ctx, userID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(userID)

	sw := newStateWriter(d)
	sw.set("id", user.ID)
	sw.set("internal_id", user.InternalID)
	sw.set("name", user.Name)
	sw.set("type", user.Type)
	sw.set("source", user.Source)
	sw.set("source_id", user.SourceID)
	sw.set("email_verified", user.EmailVerified)
	sw.set("email", user.Email)

	return sw.diags
}
