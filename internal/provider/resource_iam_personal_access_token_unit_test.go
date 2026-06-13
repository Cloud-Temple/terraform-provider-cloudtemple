package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func patList(tokens ...*client.Token) patListFunc {
	return func(ctx context.Context) ([]*client.Token, error) {
		return tokens, nil
	}
}

func patListErr(err error) patListFunc {
	return func(ctx context.Context) ([]*client.Token, error) {
		return nil, err
	}
}

func patRead(token *client.Token, err error) patReadFunc {
	return func(ctx context.Context, id string) (*client.Token, error) {
		return token, err
	}
}

// TestResolveMissingPAT pins the deletion-evidence helper: it must never
// conclude a deletion (it has no authority to), only report liveness from a
// strict listing or propagate a list failure (#281).
func TestResolveMissingPAT(t *testing.T) {
	ctx := context.Background()
	const target = "pat-1"

	t.Run("a list error fails closed and reports no live token", func(t *testing.T) {
		listErr := errors.New("boom")
		found, err := resolveMissingPAT(ctx, target, patListErr(listErr))
		if !errors.Is(err, listErr) {
			t.Fatalf("the list error must propagate, got %v", err)
		}
		if found != nil {
			t.Fatalf("a list error must not report a live token, got %v", found)
		}
	})

	t.Run("a present token is reported alive", func(t *testing.T) {
		want := &client.Token{ID: target, Name: "keep"}
		found, err := resolveMissingPAT(ctx, target, patList(&client.Token{ID: "other"}, want))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != want {
			t.Fatalf("expected the matching token, got %v", found)
		}
	})

	t.Run("an absent token is NOT a deletion", func(t *testing.T) {
		found, err := resolveMissingPAT(ctx, target, patList(&client.Token{ID: "other"}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("absence must report (nil,nil) — never a confirmed deletion, got %v", found)
		}
	})

	t.Run("an empty listing is NOT a deletion", func(t *testing.T) {
		found, err := resolveMissingPAT(ctx, target, patList())
		if err != nil || found != nil {
			t.Fatalf("empty listing must report (nil,nil), got (%v,%v)", found, err)
		}
	})

	t.Run("substring and superstring ids do not false-match", func(t *testing.T) {
		found, err := resolveMissingPAT(ctx, target, patList(
			&client.Token{ID: "pat-12"}, // superstring of pat-1
			&client.Token{ID: "pat-"},   // substring of pat-1
			&client.Token{ID: "pat"},    // substring of pat-1
		))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("only an exact id match counts, got %v", found)
		}
	})

	t.Run("nil entries are tolerated when the target is present", func(t *testing.T) {
		want := &client.Token{ID: target}
		found, err := resolveMissingPAT(ctx, target, patList(nil, want, nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != want {
			t.Fatalf("a nil entry must not hide a present token, got %v", found)
		}
	})

	t.Run("nil entries do not panic when the target is absent", func(t *testing.T) {
		found, err := resolveMissingPAT(ctx, target, patList(nil, &client.Token{ID: "other"}, nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("absence with nil entries must report (nil,nil), got %v", found)
		}
	})
}

// newPATState builds a ResourceData standing for an existing PAT in the state,
// with a secret_id that only exists locally (the API returns it only at
// creation).
func newPATState(t *testing.T) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourcePersonalAccessToken().Schema, map[string]interface{}{})
	d.SetId("pat-1")
	for k, v := range map[string]string{"name": "existing", "client_id": "pat-1", "secret_id": "super-secret"} {
		if err := d.Set(k, v); err != nil {
			t.Fatalf("seeding %q: %v", k, err)
		}
	}
	return d
}

// TestReadPATInto pins the read wiring. The overriding invariant: the resource
// is NEVER removed from the state on an inconclusive read — only a successful
// read repopulates, and the secret_id is never erased (#281).
func TestReadPATInto(t *testing.T) {
	ctx := context.Background()

	t.Run("a read error keeps the id and the secret", func(t *testing.T) {
		d := newPATState(t)
		diags := readPATInto(ctx, d, patRead(nil, errors.New("access denied")), patList())
		if !diags.HasError() {
			t.Fatal("a read error must surface as a diagnostic")
		}
		if d.Id() != "pat-1" {
			t.Fatal("a read error must never drop the resource from the state")
		}
		if got := d.Get("secret_id").(string); got != "super-secret" {
			t.Fatalf("secret_id must be preserved, got %q", got)
		}
	})

	t.Run("a successful read repopulates without erasing the secret", func(t *testing.T) {
		d := newPATState(t)
		tok := &client.Token{ID: "pat-1", Name: "fresh", Roles: []string{"r1"}, ExpirationDate: "1700000000000", Secret: ""}
		diags := readPATInto(ctx, d, patRead(tok, nil), patListErr(errors.New("list must not be called on a successful read")))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got := d.Get("name").(string); got != "fresh" {
			t.Fatalf("name must be refreshed, got %q", got)
		}
		if got := d.Get("secret_id").(string); got != "super-secret" {
			t.Fatalf("an empty Secret must not erase secret_id, got %q", got)
		}
	})

	t.Run("an inconclusive read with a failing listing fails closed", func(t *testing.T) {
		d := newPATState(t)
		diags := readPATInto(ctx, d, patRead(nil, nil), patListErr(errors.New("transient 500")))
		if !diags.HasError() {
			t.Fatal("a failing listing must fail closed with a diagnostic, never a drop")
		}
		if d.Id() != "pat-1" {
			t.Fatal("the resource must be kept in the state")
		}
		if got := d.Get("secret_id").(string); got != "super-secret" {
			t.Fatalf("secret_id must be preserved, got %q", got)
		}
	})

	t.Run("an inconclusive read recovered by the listing keeps state untouched", func(t *testing.T) {
		d := newPATState(t)
		// A listing entry with a DIFFERENT name must not be written back:
		// the recovery is liveness-only (the fields are ForceNew and the
		// listing scope is not provably the token's owner scope).
		listed := &client.Token{ID: "pat-1", Name: "FROM-LIST", Secret: ""}
		diags := readPATInto(ctx, d, patRead(nil, nil), patList(listed))
		if diags.HasError() {
			t.Fatalf("a recovered liveness must not error: %v", diags)
		}
		if d.Id() != "pat-1" {
			t.Fatal("the resource must be kept in the state")
		}
		if got := d.Get("name").(string); got != "existing" {
			t.Fatalf("liveness recovery must not overwrite fields from the listing, got %q", got)
		}
		if got := d.Get("secret_id").(string); got != "super-secret" {
			t.Fatalf("secret_id must be preserved, got %q", got)
		}
	})

	t.Run("an inconclusive read with an absent token fails closed (never drops)", func(t *testing.T) {
		d := newPATState(t)
		diags := readPATInto(ctx, d, patRead(nil, nil), patList(&client.Token{ID: "other"}))
		if !diags.HasError() {
			t.Fatal("an unconfirmed absence must fail closed, never auto-remove the resource")
		}
		if d.Id() != "pat-1" {
			t.Fatal("the resource must NOT be dropped on an unconfirmed absence")
		}
		if got := d.Get("secret_id").(string); got != "super-secret" {
			t.Fatalf("secret_id must be preserved, got %q", got)
		}
	})
}
