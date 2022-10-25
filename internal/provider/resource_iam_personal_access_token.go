package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourcePersonalAccessToken() *schema.Resource {
	return &schema.Resource{
		Description: "",

		CreateContext: resourcePersonalAccessTokenCreate,
		ReadContext:   resourcePersonalAccessTokenRead,
		DeleteContext: resourcePersonalAccessTokenDelete,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"roles": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MinItems: 1,

				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
			},
			"expiration_date": {
				Type:         schema.TypeString,
				ValidateFunc: validation.IsRFC3339Time,
				Required:     true,
				ForceNew:     true,

				// DiffSuppressOnRefresh: true,
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					o, err := time.Parse(time.RFC3339, oldValue)
					if err != nil {
						return false
					}

					n, err := time.Parse(time.RFC3339, newValue)
					if err != nil {
						return false
					}

					return n.Equal(o)
				},
			},

			// Out
			"client_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"secret_id": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func resourcePersonalAccessTokenCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	name := d.Get("name").(string)
	roles := interfaceSliceToStringSlice(d.Get("roles").([]interface{}))
	ed := d.Get("expiration_date").(string)

	parsed, err := time.Parse(time.RFC3339, ed)
	if err != nil {
		return diag.FromErr(err)
	}
	expirationDate := int(parsed.UTC().UnixMilli())

	token, err := client.IAM().PAT().Create(ctx, name, roles, expirationDate)
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, token.ID)
	sw.set("client_id", token.ID)
	sw.set("secret_id", token.Secret)

	return sw.diags
}

func resourcePersonalAccessTokenRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	token, err := client.IAM().PAT().Read(ctx, d.Id())
	if err != nil {
		// TODO: detect properly that the token has been removed
		return nil
		// return diag.FromErr(err)
	}

	i, err := strconv.ParseInt(token.ExpirationDate, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	expirationDate := time.Unix(i/1000, i%1000)

	sw := newStateWriter(d, token.ID)
	sw.set("name", token.Name)
	sw.set("roles", stringSliceToInterfaceSlice(token.Roles))
	sw.set("expiration_date", expirationDate.Format(time.RFC3339))
	sw.set("client_id", token.ID)
	sw.set("secret_id", token.Secret)

	return sw.diags
}

func resourcePersonalAccessTokenDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	err := client.IAM().PAT().Delete(ctx, d.Id())
	return diag.FromErr(err)
}
