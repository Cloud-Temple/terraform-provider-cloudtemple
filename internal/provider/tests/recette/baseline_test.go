package recette

import (
	"context"
	"sort"
	"strings"
	"testing"
)

// Destroy-to-empty proof (#300 D2.5). The recette tenant is not assumed to be
// literally empty: it may carry pre-existing platform/substrate objects. So we
// capture the PRE-BUILD baseline set of ids/names, then after the SDK
// auto-destroy assert the listing returns EXACTLY to that baseline (no orphan
// left, nothing pre-existing wrongly deleted).
//
// We use the *Strict listings (HTTP 200 only): a 206 partial listing cannot
// prove an absence, so it must fail closed rather than produce a false
// "destroyed-to-empty".
//
// acl_entry has no strict list endpoint -> its destroy-to-empty is a documented
// residual risk, verified indirectly: revoking via the bucket teardown and the
// bucket/account baselines returning to their pre-build set.

type stringSet map[string]struct{}

// patSet keys PATs by id, restricted at capture time to the authenticated
// principal's own tokens so the baseline is comparable to the post-destroy set.
type patSet map[string]struct{}

func newStringSet(items []string) stringSet {
	s := make(stringSet, len(items))
	for _, it := range items {
		s[it] = struct{}{}
	}
	return s
}

// diff returns elements in have that are not in want (orphans) and elements in
// want missing from have (wrongly deleted), as sorted slices for stable output.
func diffSets(want, have stringSet) (added, missing []string) {
	for k := range have {
		if _, ok := want[k]; !ok {
			added = append(added, k)
		}
	}
	for k := range want {
		if _, ok := have[k]; !ok {
			missing = append(missing, k)
		}
	}
	sort.Strings(added)
	sort.Strings(missing)
	return added, missing
}

func mustCaptureOSBaselines(t *testing.T) (buckets stringSet, accounts stringSet) {
	t.Helper()
	ctx := context.Background()
	c, err := newRecetteClient()
	if err != nil {
		t.Fatalf("recette baseline: %s", err)
	}
	bs, err := c.ObjectStorage().Bucket().ListStrict(ctx)
	if err != nil {
		t.Fatalf("recette baseline: failed to list buckets: %s", err)
	}
	as, err := c.ObjectStorage().StorageAccount().ListStrict(ctx)
	if err != nil {
		t.Fatalf("recette baseline: failed to list storage accounts: %s", err)
	}
	var bnames, anames []string
	for _, b := range bs {
		if b != nil {
			bnames = append(bnames, b.Name)
		}
	}
	for _, a := range as {
		if a != nil {
			anames = append(anames, a.Name)
		}
	}
	return newStringSet(bnames), newStringSet(anames)
}

func assertOSBaselinesRestored(t *testing.T, buckets, accounts stringSet) error {
	t.Helper()
	ctx := context.Background()
	c, err := newRecetteClient()
	if err != nil {
		return err
	}
	bs, err := c.ObjectStorage().Bucket().ListStrict(ctx)
	if err != nil {
		return err
	}
	as, err := c.ObjectStorage().StorageAccount().ListStrict(ctx)
	if err != nil {
		return err
	}
	var bnames, anames []string
	for _, b := range bs {
		if b != nil {
			bnames = append(bnames, b.Name)
		}
	}
	for _, a := range as {
		if a != nil {
			anames = append(anames, a.Name)
		}
	}
	if err := assertRestored("bucket", buckets, newStringSet(bnames)); err != nil {
		return err
	}
	return assertRestored("storage account", accounts, newStringSet(anames))
}

func mustCapturePATBaseline(t *testing.T) patSet {
	t.Helper()
	ctx := context.Background()
	c, err := newRecetteClient()
	if err != nil {
		t.Fatalf("recette baseline: %s", err)
	}
	lt, err := c.Token(ctx)
	if err != nil {
		t.Fatalf("recette baseline: failed to authenticate: %s", err)
	}
	tokens, err := c.IAM().PAT().ListStrict(ctx)
	if err != nil {
		t.Fatalf("recette baseline: failed to list PATs: %s", err)
	}
	s := make(patSet)
	for _, tok := range tokens {
		if tok == nil {
			continue
		}
		// Restrict to the principal's own tokens so the set is comparable
		// after destroy (the listing is not scoped server-side, #226).
		if tok.UserId == lt.UserID() && tok.TenantId == lt.TenantID() {
			s[tok.ID] = struct{}{}
		}
	}
	return s
}

func assertPATBaselineRestored(t *testing.T, baseline patSet) error {
	t.Helper()
	ctx := context.Background()
	c, err := newRecetteClient()
	if err != nil {
		return err
	}
	lt, err := c.Token(ctx)
	if err != nil {
		return err
	}
	tokens, err := c.IAM().PAT().ListStrict(ctx)
	if err != nil {
		return err
	}
	have := make(stringSet)
	for _, tok := range tokens {
		if tok == nil {
			continue
		}
		if tok.UserId == lt.UserID() && tok.TenantId == lt.TenantID() {
			have[tok.ID] = struct{}{}
		}
	}
	return assertRestored("personal access token", stringSet(baseline), have)
}

// assertRestored compares the post-destroy set to the pre-build baseline. It
// reports orphans (added) and wrongly-removed pre-existing objects (missing).
// It never prints a secret; ids/names are non-sensitive here.
func assertRestored(kind string, baseline, have stringSet) error {
	added, missing := diffSets(baseline, have)
	if len(added) == 0 && len(missing) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(kind + " set did not return to the pre-build baseline after destroy")
	if len(added) > 0 {
		b.WriteString("; orphans left behind: " + strings.Join(added, ", "))
	}
	if len(missing) > 0 {
		b.WriteString("; pre-existing items wrongly removed: " + strings.Join(missing, ", "))
	}
	return errString(b.String())
}

// errString is a tiny error type so assertRestored stays allocation-light and
// dependency-free.
type errString string

func (e errString) Error() string { return string(e) }
