package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
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
	c := getClient(meta)

	name := d.Get("name").(string)
	roles := interfaceSliceToStringSlice(d.Get("roles").([]interface{}))
	ed := d.Get("expiration_date").(string)

	parsed, err := time.Parse(time.RFC3339, ed)
	if err != nil {
		return diag.Errorf("failed to parse personal access token expiration_date: %s", err)
	}
	expirationDate := int(parsed.UTC().UnixMilli())

	token, err := c.IAM().PAT().Create(ctx, name, roles, expirationDate)
	if err != nil {
		return diag.Errorf("failed to create personal access token: %s", err)
	}

	sw := newStateWriter(d)
	sw.set("client_id", token.ID)
	sw.set("secret_id", token.Secret)
	d.SetId(token.ID)

	return sw.diags
}

// patReadFunc and patListFunc abstract the PAT API surface used by
// readPATInto so the read logic is unit tested without HTTP calls.
type patReadFunc func(ctx context.Context, id string) (*client.Token, error)
type patListFunc func(ctx context.Context) ([]*client.Token, error)

// resolveMissingPAT is called when the per-id read returned nil. The IAM API
// maps BOTH an absent token AND a transient HTTP 500 to nil (see client
// iam_pat.go), so a nil read never proves a deletion. This helper only looks
// for the id in a strict, complete listing and reports liveness; it never
// concludes a deletion on its own (#281).
//
//   - list error (includes a non-200 answer rejected by ListStrict) -> the
//     caller fails closed and keeps the resource in the state;
//   - id present -> the token is alive (the read 500 was transient);
//   - id absent  -> the caller fails closed too: the listing scope is not
//     provably equal to the token's ownership scope, so absence is NOT
//     deletion evidence and the resource is never auto-removed.
//
// nil entries in the listing are tolerated (a JSON [null] decodes to one).
func resolveMissingPAT(ctx context.Context, id string, list patListFunc) (*client.Token, error) {
	tokens, err := list(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range tokens {
		if t != nil && t.ID == id {
			return t, nil
		}
	}
	return nil, nil
}

func resourcePersonalAccessTokenRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return readPATInto(ctx, d, c.IAM().PAT().Read, c.IAM().PAT().ListStrict)
}

// readPATInto holds the testable read logic of the PAT resource. The client's
// Terraform state is the absolute priority: the resource is NEVER removed from
// the state on an inconclusive read. An unconfirmed read fails closed (returns
// a diagnostic and leaves the state untouched) instead of dropping the id.
func readPATInto(ctx context.Context, d *schema.ResourceData, read patReadFunc, list patListFunc) diag.Diagnostics {
	token, err := read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read personal access token: %s", err)
	}
	if token == nil {
		// The per-id read is inconclusive (the API maps both an absent token
		// and a transient 500 to nil). We never drop the resource on this
		// signal alone — see resolveMissingPAT.
		found, lerr := resolveMissingPAT(ctx, d.Id(), list)
		if lerr != nil {
			return diag.Errorf(
				"could not confirm whether personal access token %s still exists: the read was inconclusive and the token listing failed; the resource is left untouched in the state to avoid a wrong deletion: %s",
				d.Id(), lerr,
			)
		}
		if found == nil {
			return diag.Errorf(
				"personal access token %s is no longer returned by the API and its deletion could not be confirmed; the resource is kept in the state. If you deleted it intentionally, remove it with `terraform state rm <address>`.",
				d.Id(),
			)
		}
		// found != nil: the token is alive (the read was a transient 500). Its
		// attributes are ForceNew (immutable) and we cannot prove the listing
		// is scoped to this token's owner, so we recover liveness only and
		// keep the existing state untouched — we write nothing from the list.
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
	// The API only returns the secret at creation: overwriting it on
	// refresh would erase the only copy from the state (#264 plan, Lot D).
	if token.Secret != "" {
		sw.set("secret_id", token.Secret)
	}

	return sw.diags
}

func resourcePersonalAccessTokenDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	if err := c.IAM().PAT().Delete(ctx, d.Id()); err != nil {
		return diag.Errorf("failed to delete personal access token: %s", err)
	}
	return nil
}
