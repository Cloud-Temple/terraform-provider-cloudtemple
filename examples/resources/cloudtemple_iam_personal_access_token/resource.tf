resource "cloudtemple_iam_personal_access_token" "example" {
  name            = "this is a test of Terraform"
  expiration_date = "2022-10-20T00:00:00Z"
  roles = [
    "b7a0d310-839e-419b-9e56-fb052eb17958"
  ]
}

output "token" {
  value     = cloudtemple_iam_personal_access_token.example
  sensitive = true
}