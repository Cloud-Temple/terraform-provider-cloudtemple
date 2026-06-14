package recette

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// recetteNamePrefix scopes every destructive sweep. Only resources whose name
// carries this prefix (or the created_by=Terraform tag) are ever deleted.
const recetteNamePrefix = "test-terraform"

// createdByTag identifies Terraform-managed resources via tag.
const (
	createdByTagKey   = "created_by"
	createdByTagValue = "Terraform"
)

// init registers the per-type sweepers. They run ONLY via the -sweep flag and
// ONLY because TestMain wires resource.TestMain(m). AddTestSweepers does NOT
// auto-run on any test start/end; this registration alone deletes nothing.
//
// Every sweeper re-asserts the tenant guard before touching anything, because
// -sweep bypasses the TestCase PreCheck.
func init() {
	resource.AddTestSweepers("cloudtemple_object_storage_bucket", &resource.Sweeper{
		Name: "cloudtemple_object_storage_bucket",
		F:    func(string) error { return sweepRecette(context.Background()) },
	})
	resource.AddTestSweepers("cloudtemple_object_storage_storage_account", &resource.Sweeper{
		Name:         "cloudtemple_object_storage_storage_account",
		Dependencies: []string{"cloudtemple_object_storage_bucket"},
		F:            func(string) error { return nil }, // body shared via sweepRecette
	})
	resource.AddTestSweepers("cloudtemple_iam_personal_access_token", &resource.Sweeper{
		Name: "cloudtemple_iam_personal_access_token",
		F:    func(string) error { return nil }, // body shared via sweepRecette
	})
}

// sweepRecette is the single guarded teardown body. It is called BOTH from the
// recette TestMain at start-of-run (clean slate) and from the registered bucket
// sweeper under -sweep. It is idempotent and re-asserts the tenant guard first.
//
// Scope discipline:
//   - buckets / storage accounts: deleted only if the name has the recette
//     prefix OR carries the created_by=Terraform tag;
//   - PATs: additionally filtered to the authenticated principal's own UserId
//     AND TenantId, so a shared listing can never delete another principal's
//     token.
//
// acl_entry is NOT swept directly (no list endpoint, no importer); revoking the
// bucket removes its grants.
//
// SAFETY: this function NEVER prints a tenant id, a secret, or a resource id at
// a level that could leak; errors reference resource NAMES only, which are
// recette-prefixed by construction.
func sweepRecette(ctx context.Context) error {
	// Re-assert the guard: -sweep bypasses the TestCase PreCheck, and the
	// start-of-run call must never trust an unverified tenant.
	if err := guardLiveTenant(ctx); err != nil {
		return err
	}

	c, err := newRecetteClient()
	if err != nil {
		return err
	}

	lt, err := c.Token(ctx)
	if err != nil {
		return fmt.Errorf("recette sweep: failed to authenticate: %w", err)
	}

	if err := sweepBuckets(ctx, c); err != nil {
		return err
	}
	if err := sweepStorageAccounts(ctx, c); err != nil {
		return err
	}
	if err := sweepPATs(ctx, c, lt); err != nil {
		return err
	}
	return nil
}

// isRecetteScoped reports whether a name/tags pair is in the recette deletion
// scope: recette name prefix OR the created_by=Terraform tag.
func isRecetteScoped(name string, hasCreatedByTag bool) bool {
	return strings.HasPrefix(name, recetteNamePrefix) || hasCreatedByTag
}

func sweepBuckets(ctx context.Context, c *client.Client) error {
	buckets, err := c.ObjectStorage().Bucket().ListStrict(ctx)
	if err != nil {
		return fmt.Errorf("recette sweep: failed to list buckets: %w", err)
	}
	for _, b := range buckets {
		if b == nil {
			continue
		}
		hasTag := false
		for _, tag := range b.Tags {
			if tag.Key == createdByTagKey && tag.Value == createdByTagValue {
				hasTag = true
				break
			}
		}
		if !isRecetteScoped(b.Name, hasTag) {
			continue
		}
		activityID, err := c.ObjectStorage().Bucket().Delete(ctx, b.Name)
		if err != nil {
			return fmt.Errorf("recette sweep: failed to delete bucket %s: %w", b.Name, err)
		}
		if _, err := c.Activity().WaitForCompletion(ctx, activityID, nil); err != nil {
			return fmt.Errorf("recette sweep: failed to confirm bucket %s deletion: %w", b.Name, err)
		}
	}
	return nil
}

func sweepStorageAccounts(ctx context.Context, c *client.Client) error {
	accounts, err := c.ObjectStorage().StorageAccount().ListStrict(ctx)
	if err != nil {
		return fmt.Errorf("recette sweep: failed to list storage accounts: %w", err)
	}
	for _, a := range accounts {
		if a == nil {
			continue
		}
		hasTag := false
		for _, tag := range a.Tags {
			if tag.Key == createdByTagKey && tag.Value == createdByTagValue {
				hasTag = true
				break
			}
		}
		if !isRecetteScoped(a.Name, hasTag) {
			continue
		}
		activityID, err := c.ObjectStorage().StorageAccount().Delete(ctx, a.Name)
		if err != nil {
			return fmt.Errorf("recette sweep: failed to delete storage account %s: %w", a.Name, err)
		}
		if _, err := c.Activity().WaitForCompletion(ctx, activityID, nil); err != nil {
			return fmt.Errorf("recette sweep: failed to confirm storage account %s deletion: %w", a.Name, err)
		}
	}
	return nil
}

func sweepPATs(ctx context.Context, c *client.Client, lt *client.LoginToken) error {
	tokens, err := c.IAM().PAT().ListStrict(ctx)
	if err != nil {
		return fmt.Errorf("recette sweep: failed to list personal access tokens: %w", err)
	}
	for _, t := range tokens {
		if t == nil {
			continue
		}
		// PAT().List is not scoped server-side (#226): restrict the destructive
		// sweep to the authenticated principal's own tokens AND the recette
		// name prefix. PATs carry no created_by tag, so the prefix is the only
		// name-scope signal here.
		if !strings.HasPrefix(t.Name, recetteNamePrefix) {
			continue
		}
		if t.UserId != lt.UserID() || t.TenantId != lt.TenantID() {
			continue
		}
		if err := c.IAM().PAT().Delete(ctx, t.ID); err != nil {
			return fmt.Errorf("recette sweep: failed to delete personal access token %s: %w", t.Name, err)
		}
	}
	return nil
}
