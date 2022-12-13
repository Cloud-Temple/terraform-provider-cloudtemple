# Read a token using its ID
data "cloudtemple_iam_personal_access_token" "id" {
  id = "b0232f77-54eb-49d2-abea-39b312db42c5"
}

# Read a token using its name
data "cloudtemple_iam_personal_access_token" "name" {
  name = "Terraform"
}
