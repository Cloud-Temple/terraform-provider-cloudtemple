package recette

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Phase 1 recette lifecycle harness — PAT + object storage only.
//
// Substrate assumption (the Phase 1 prerequisite, see #300 Q1): even an "empty"
// recette tenant must already have, pre-existing and datasource-only:
//   - one IAM role (referenced by name/id to create the PAT — PAT Create
//     HARD-FAILS with no role);
//   - one object-storage role (referenced BY NAME by the acl_entry);
//   - the object-storage service/namespace enabled;
//   - credentials with object-storage + activity read/wait permissions.
//
// These are injected by env var NAMES only (never values committed):
//   - CLOUDTEMPLE_RECETTE_IAM_ROLE_NAME   : IAM role name for the PAT
//   - CLOUDTEMPLE_RECETTE_OS_ROLE_NAME    : object-storage role name for the ACL
//
// EVERY recette TestCase PreCheck calls recettePreCheck, which re-asserts the
// tenant guard. The guard's real un-skippability comes from TestMain (it runs
// the auth+tenant assertion before any TestCase), not from this PreCheck.

const (
	recetteIAMRoleNameEnv = "CLOUDTEMPLE_RECETTE_IAM_ROLE_NAME"
	recetteOSRoleNameEnv  = "CLOUDTEMPLE_RECETTE_OS_ROLE_NAME"
)

// recettePreCheck runs before each TestCase. With TF_ACC unset the SDK skips
// resource.Test entirely, so this never runs without live mode. It validates
// the required substrate env vars and re-asserts the tenant guard (defence in
// depth on top of the TestMain gate).
func recettePreCheck(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		recetteTenantEnvName,
		recetteIAMRoleNameEnv,
		recetteOSRoleNameEnv,
	} {
		if os.Getenv(name) == "" {
			t.Fatalf("%s must be set to run the recette harness", name)
		}
	}
	if err := guardLiveTenant(t.Context()); err != nil {
		// Redacted by construction.
		t.Fatalf("recette tenant guard refused the run: %s", err)
	}
}

// captureAttr stores the value of a state attribute into *dst so a later step
// can assert the EXACT same value is preserved across a refresh.
func captureAttr(resourceName, attr string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		v, ok := rs.Primary.Attributes[attr]
		if !ok {
			return fmt.Errorf("attribute %s not set on %s", attr, resourceName)
		}
		if v == "" {
			return fmt.Errorf("attribute %s on %s is empty; expected a sensitive value", attr, resourceName)
		}
		*dst = v
		return nil
	}
}

// assertAttrUnchanged asserts a previously captured value is still EXACTLY
// present (not merely set). This is the secret-preservation-on-refresh proof.
func assertAttrUnchanged(resourceName, attr string, want *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		got := rs.Primary.Attributes[attr]
		if got != *want {
			// Never print the secret VALUE: report only that it changed.
			return fmt.Errorf("attribute %s on %s was not preserved across refresh (value changed)", attr, resourceName)
		}
		return nil
	}
}

// TestAccRecettePAT exercises the PAT lifecycle:
//   - apply + convergence (empty second plan: SDK built-in);
//   - read-correctness via ImportState/ImportStateVerify (secret_id ignored,
//     since the API never returns the secret after creation);
//   - secret preservation: capture secret_id after apply, run a refresh-only
//     plan, assert the EXACT same secret_id is still present;
//   - destroy (SDK auto-destroy) + destroy-to-empty via a PAT ListStrict
//     baseline captured before build.
func TestAccRecettePAT(t *testing.T) {
	var capturedSecret string
	var patBaseline patSet

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { recettePreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				// Capture the pre-build baseline before creating anything.
				Config: testAccRecettePATConfig(),
				PreConfig: func() {
					patBaseline = mustCapturePATBaseline(t)
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_iam_personal_access_token.recette", "client_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_iam_personal_access_token.recette", "secret_id"),
					captureAttr("cloudtemple_iam_personal_access_token.recette", "secret_id", &capturedSecret),
				),
			},
			{
				// Refresh-only plan: prove the sensitive secret_id is preserved
				// (not blanked) and the plan is empty (no config-driven change).
				Config:             testAccRecettePATConfig(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					assertAttrUnchanged("cloudtemple_iam_personal_access_token.recette", "secret_id", &capturedSecret),
				),
			},
			{
				// Read-correctness: import round-trip. The API never returns the
				// secret after creation, so secret_id is ignored on verify.
				ResourceName:            "cloudtemple_iam_personal_access_token.recette",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret_id"},
			},
		},
		// Destroy-to-empty: after the SDK auto-destroy, the PAT listing must
		// return to the pre-build baseline.
		CheckDestroy: func(s *terraform.State) error {
			return assertPATBaselineRestored(t, patBaseline)
		},
	})
}

// TestAccRecetteObjectStorage exercises the object-storage stack:
//   - bucket (private, NO inline acl_entry) + storage_account + standalone
//     acl_entry (bucket x os-role-by-name x storage_account);
//   - convergence: a second plan must be empty (SDK built-in). See the
//     bucket/acl drift note below.
//   - read-correctness: ImportState/ImportStateVerify for bucket and
//     storage_account; acl_entry has NO importer, so its state is asserted via
//     Check (composite id) instead.
//   - secret preservation: capture access_secret_key after apply, refresh-only
//     plan, assert the exact same key is still present.
//   - destroy + destroy-to-empty via bucket/account ListStrict baselines.
//
// BUCKET/ACL DRIFT (the Phase 1 convergence risk, #300 D2.1):
// The bucket resource's Read re-populates its INLINE acl_entry Optional field
// from the live ACL listing (resource_object_storage_bucket.go Read). The
// standalone acl_entry resource grants the same ACL. If the bucket Read writes
// a non-empty inline acl_entry into the bucket state while the bucket CONFIG
// declares none, the SDK will see a permanent diff on the bucket and the second
// plan will NOT be empty.
//
// This test deliberately keeps the bucket config free of any inline acl_entry
// and relies on the standalone resource only. It does NOT set
// ExpectNonEmptyPlan: papering over the drift would hide a genuine state-safety
// finding. If this TestCase fails its convergence step in live mode with a
// bucket acl_entry diff, that is a provider finding to report (do not fix here),
// per the brief.
func TestAccRecetteObjectStorage(t *testing.T) {
	var capturedKey string
	var bucketBaseline stringSet
	var accountBaseline stringSet

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { recettePreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecetteObjectStorageConfig(),
				PreConfig: func() {
					bucketBaseline, accountBaseline = mustCaptureOSBaselines(t)
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_bucket.recette", "id"),
					resource.TestCheckResourceAttr("cloudtemple_object_storage_bucket.recette", "access_type", "private"),
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_storage_account.recette", "access_key_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_storage_account.recette", "access_secret_key"),
					captureAttr("cloudtemple_object_storage_storage_account.recette", "access_secret_key", &capturedKey),
					// acl_entry has no importer; assert its composite id + parts.
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_acl_entry.recette", "id"),
					resource.TestCheckResourceAttrPair(
						"cloudtemple_object_storage_acl_entry.recette", "bucket",
						"cloudtemple_object_storage_bucket.recette", "name",
					),
					resource.TestCheckResourceAttrPair(
						"cloudtemple_object_storage_acl_entry.recette", "storage_account",
						"cloudtemple_object_storage_storage_account.recette", "name",
					),
				),
			},
			{
				// Refresh-only: convergence + storage-account secret preservation.
				Config:             testAccRecetteObjectStorageConfig(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					assertAttrUnchanged("cloudtemple_object_storage_storage_account.recette", "access_secret_key", &capturedKey),
				),
			},
			{
				ResourceName:      "cloudtemple_object_storage_bucket.recette",
				ImportState:       true,
				ImportStateVerify: true,
				// access_secret_key/access_key_id live on the storage account, not
				// the bucket; the bucket Read re-derives acl_entry, so ignore it on
				// verify to avoid coupling the import check to the drift question.
				ImportStateVerifyIgnore: []string{"acl_entry"},
			},
			{
				ResourceName:      "cloudtemple_object_storage_storage_account.recette",
				ImportState:       true,
				ImportStateVerify: true,
				// The secret is only returned at creation; it is absent from a
				// fresh import read.
				ImportStateVerifyIgnore: []string{"access_secret_key"},
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			return assertOSBaselinesRestored(t, bucketBaseline, accountBaseline)
		},
	})
}

func testAccRecettePATConfig() string {
	return fmt.Sprintf(`
data "cloudtemple_iam_role" "recette" {
  name = %q
}

resource "cloudtemple_iam_personal_access_token" "recette" {
  name            = "test-terraform-recette-pat"
  roles           = [data.cloudtemple_iam_role.recette.id]
  expiration_date = "2999-01-01T00:00:00Z"
}
`, os.Getenv(recetteIAMRoleNameEnv))
}

func testAccRecetteObjectStorageConfig() string {
	return fmt.Sprintf(`
data "cloudtemple_object_storage_role" "recette" {
  name = %q
}

resource "cloudtemple_object_storage_bucket" "recette" {
  name        = "test-terraform-recette-bucket"
  access_type = "private"
  # Deliberately NO inline acl_entry block: the ACL is managed by the
  # standalone resource below. See the bucket/acl drift note on the test.
}

resource "cloudtemple_object_storage_storage_account" "recette" {
  name = "test-terraform-recette-sa"
}

resource "cloudtemple_object_storage_acl_entry" "recette" {
  bucket          = cloudtemple_object_storage_bucket.recette.name
  role            = data.cloudtemple_object_storage_role.recette.name
  storage_account = cloudtemple_object_storage_storage_account.recette.name
}
`, os.Getenv(recetteOSRoleNameEnv))
}
