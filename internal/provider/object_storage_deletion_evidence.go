package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// bucketNames projects a bucket listing into the names used as identity for
// reads, skipping nil entries (a JSON `[null]` decodes into one).
func bucketNames(buckets []*client.Bucket) []string {
	names := make([]string, 0, len(buckets))
	for _, b := range buckets {
		if b != nil {
			names = append(names, b.Name)
		}
	}
	return names
}

// storageAccountNames projects a storage account listing into the names used as
// identity for reads, skipping nil entries.
func storageAccountNames(accounts []*client.StorageAccount) []string {
	names := make([]string, 0, len(accounts))
	for _, a := range accounts {
		if a != nil {
			names = append(names, a.Name)
		}
	}
	return names
}

// confirmObjectStorageOrKeep handles the nil-read branch of an object storage
// resource's Read. It ALWAYS returns an error diagnostic and NEVER touches the
// ResourceData, so it is structurally incapable of dropping the resource: the
// resource is kept in the state in every case, and the read never succeeds on
// an unreadable resource.
//
// A nil per-id read is inconclusive (the client maps HTTP 403 to nil). The
// object storage listings are unscoped (the caller's buckets / accounts) and
// their completeness is not provable — a forbidden-but-existing object can be
// omitted from a 200 ("accessible only"), and a 206 is partial — so absence is
// NOT deletion evidence. The resource is therefore never auto-removed; and a
// still-listed resource is also not reported as a successful refresh, because
// its (mutable) attributes could not be re-read and the state may be stale. The
// listing only sharpens the diagnostic.
//
// name is the resource's stable name (its identity for reads); kind is the
// human label. Empty entries in the listing are skipped; matching is exact.
func confirmObjectStorageOrKeep(ctx context.Context, name, kind string, list func(ctx context.Context) ([]string, error)) diag.Diagnostics {
	names, err := list(ctx)
	if err != nil {
		return diag.Errorf(
			"%s %q could not be read and its existence could not be confirmed (the listing failed); the resource is kept in the state to avoid a wrong deletion: %s",
			kind, name, err,
		)
	}
	for _, candidate := range names {
		if candidate != "" && candidate == name {
			return diag.Errorf(
				"%s %q could not be read but is still listed; the resource is kept in the state (refusing to drop it on a likely transient error or access restriction). Its attributes could not be refreshed — retry once the read succeeds.",
				kind, name,
			)
		}
	}
	return diag.Errorf(
		"%s %q is no longer returned by the API and is not in the listing; its deletion could not be confirmed (it may have been deleted, or your access may have changed). The resource is kept in the state. If you removed it intentionally, run `terraform state rm` on it.",
		kind, name,
	)
}
