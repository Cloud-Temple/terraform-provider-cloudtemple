# Phase 1 recette stack — PAT + object storage (INERT).
#
# This stack is for the LATER supervised phase (Phase B in the #300 design); it
# is NOT run by the Go harness and NOT run automatically. A human runs it under
# explicit GO, via run.sh, which guards the tenant and always destroys.
#
# Scope: exactly the Phase 1 resources buildable on a TRULY EMPTY tenant with NO
# compute substrate:
#   - cloudtemple_iam_personal_access_token (needs a pre-existing IAM role)
#   - cloudtemple_object_storage_bucket
#   - cloudtemple_object_storage_storage_account
#   - cloudtemple_object_storage_acl_entry (bucket x os-role-by-name x account)
#
# EXCLUDED (per the brief): all compute (Q1 substrate unresolved) and
# global_access_key (cannot be deleted via API).

terraform {
  required_providers {
    cloudtemple = {
      source = "cloud-temple/cloudtemple"
    }
  }
}

# Credentials come from the standard CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID
# environment variables (sourced from a gitignored .env.recette, never committed).
provider "cloudtemple" {}

# Pre-existing IAM role for the PAT. PAT create HARD-FAILS with no role, so this
# role must already exist on the tenant (datasource-only).
data "cloudtemple_iam_role" "recette" {
  name = var.iam_role_name
}

# Pre-existing object-storage role for the ACL. The acl_entry references the
# role BY NAME.
data "cloudtemple_object_storage_role" "recette" {
  name = var.object_storage_role_name
}

resource "cloudtemple_iam_personal_access_token" "recette" {
  name            = "test-terraform-recette-pat"
  roles           = [data.cloudtemple_iam_role.recette.id]
  expiration_date = var.pat_expiration_date
}

resource "cloudtemple_object_storage_bucket" "recette" {
  name        = "test-terraform-recette-bucket"
  access_type = "private"

  # Deliberately NO inline acl_entry block. The ACL is managed by the standalone
  # resource below. Managing the same ACL both inline (on the bucket) and via the
  # standalone resource causes convergence drift, because the bucket Read
  # re-populates its inline acl_entry from the live ACL listing.
}

resource "cloudtemple_object_storage_storage_account" "recette" {
  name = "test-terraform-recette-sa"
}

resource "cloudtemple_object_storage_acl_entry" "recette" {
  bucket          = cloudtemple_object_storage_bucket.recette.name
  role            = data.cloudtemple_object_storage_role.recette.name
  storage_account = cloudtemple_object_storage_storage_account.recette.name
}
