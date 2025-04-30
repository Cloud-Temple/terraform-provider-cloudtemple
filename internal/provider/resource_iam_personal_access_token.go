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
		Description: "Create and manage personal access tokens for a user. " +
			"Personal access tokens are used to authenticate API requests. " +
			"Tokens are valid until the specified expiration date. " +
			"Tokens can be created with specific roles, which define the permissions associated with the token. " +
			"Tokens can be used to authenticate API requests on behalf of the user who created them. " +
			"Tokens can be revoked at any time, which will invalidate them and prevent further use.",

		CreateContext: resourcePersonalAccessTokenCreate,
		ReadContext:   resourcePersonalAccessTokenRead,
		DeleteContext: resourcePersonalAccessTokenDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

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
		return diag.Errorf("failed to parse personal access token expiration_date: %s", err)
	}
	expirationDate := int(parsed.UTC().UnixMilli())

	token, err := client.IAM().PAT().Create(ctx, name, roles, expirationDate)
	if err != nil {
		return diag.Errorf("failed to create personal access token: %s", err)
	}

	sw := newStateWriter(d)
	sw.set("client_id", token.ID)
	sw.set("secret_id", token.Secret)
	d.SetId(token.ID)

	return sw.diags
}

func resourcePersonalAccessTokenRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	token, err := client.IAM().PAT().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read personal access token: %s", err)
	}
	if token == nil {
		d.SetId("")
		return nil
	}

	i, err := strconv.ParseInt(token.ExpirationDate, 10, 64)
	if err != nil {
		return diag.Errorf("failed to parse token expiration date: %s", err)
	}
	expirationDate := time.Unix(i/1000, i%1000)

	sw := newStateWriter(d)
	sw.set("name", token.Name)
	sw.set("roles", stringSliceToInterfaceSlice(token.Roles))
	sw.set("expiration_date", expirationDate.Format(time.RFC3339))
	sw.set("client_id", token.ID)
	sw.set("secret_id", token.Secret)

	return sw.diags
}

func resourcePersonalAccessTokenDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	if err := client.IAM().PAT().Delete(ctx, d.Id()); err != nil {
		return diag.Errorf("failed to delete personal access token: %s", err)
	}
	return nil
}
