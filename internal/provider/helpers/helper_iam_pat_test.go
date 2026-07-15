package helpers

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestFlattenTokenNeverEmitsTheSecret is the dedicated state-secret test for
// the PAT flatten layer. The client Token struct carries a create-only Secret
// (the API only returns it once, at creation). FlattenToken feeds the PAT
// DATASOURCES, whose schemas declare NO secret field at all. Emitting the
// secret from the flatten layer would be doubly wrong:
//
//   - it would leak the secret into a freely-readable datasource state;
//   - it would emit a key the datasource schema does not declare, so d.Set
//     would fail with "Invalid address to set" and break the read (#243).
//
// The resource keeps the secret through a SEPARATE path (the resource Read
// guards `if token.Secret != ""` before writing `secret_id`), not through
// FlattenToken. This test pins that FlattenToken stays secret-free even when
// the source Token is fully populated with a secret.
func TestFlattenTokenNeverEmitsTheSecret(t *testing.T) {
	token := &client.Token{
		ID:             "tok-1",
		Name:           "ci-token",
		Secret:         "super-secret-value",
		Roles:          []string{"role-a", "role-b"},
		ExpirationDate: "1700000000000",
		UserId:         "user-1",
		TenantId:       "tenant-1",
		TenantName:     "Tenant One",
	}

	got := FlattenToken(token)

	// No key may carry the secret, by name or by value.
	for _, forbidden := range []string{"secret", "secret_id", "Secret"} {
		if _, present := got[forbidden]; present {
			t.Errorf("FlattenToken emits forbidden secret key %q; it would leak the secret into a readable datasource and break d.Set (#243)", forbidden)
		}
	}
	for k, v := range got {
		if s, ok := v.(string); ok && s == token.Secret {
			t.Errorf("FlattenToken emits the secret value under key %q; the create-only secret must never reach the flatten output", k)
		}
	}

	// The non-secret attributes are preserved and correct.
	assertEq(t, "id", got["id"], "tok-1")
	assertEq(t, "name", got["name"], "ci-token")
	assertEq(t, "expiration_date", got["expiration_date"], "1700000000000")
	assertEq(t, "user_id", got["user_id"], "user-1")
	assertEq(t, "tenant_id", got["tenant_id"], "tenant-1")
	assertEq(t, "tenant_name", got["tenant_name"], "Tenant One")

	roles, ok := got["roles"].([]string)
	if !ok {
		t.Fatalf("roles has type %T, want []string", got["roles"])
	}
	if len(roles) != 2 || roles[0] != "role-a" || roles[1] != "role-b" {
		t.Errorf("roles = %v, want [role-a role-b] in order", roles)
	}
}

// TestFlattenTokenKeySetMatchesDatasource pins the exact emitted key set
// against the PAT datasource contract: the helper must emit only id, name,
// roles, expiration_date, user_id, tenant_id, tenant_name. An extra key (the
// secret being the dangerous one) breaks the datasource read.
func TestFlattenTokenKeySetMatchesDatasource(t *testing.T) {
	got := FlattenToken(&client.Token{ID: "x", Secret: "s"})
	want := map[string]bool{
		"id": true, "name": true, "roles": true, "expiration_date": true,
		"user_id": true, "tenant_id": true, "tenant_name": true,
	}
	for k := range got {
		if !want[k] {
			t.Errorf("FlattenToken emits undeclared key %q (the PAT datasource schema would reject it)", k)
		}
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("FlattenToken is missing the expected key %q", k)
		}
	}
}
