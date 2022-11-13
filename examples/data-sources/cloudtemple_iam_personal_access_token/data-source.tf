# Read a token using its ID
data "cloudtemple_iam_personal_access_token" "id" {
  id = "6f0ac881-bb3d-4c0b-8276-d38f71aa392d"
}

# Read a token using its name
data "cloudtemple_iam_personal_access_token" "name" {
  name = "Terraform"
}