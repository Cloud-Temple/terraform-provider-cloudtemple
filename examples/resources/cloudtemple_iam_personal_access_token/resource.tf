resource "cloudtemple_iam_personal_access_token" "example" {
  name            = "this is a test of Terraform"
  expiration_date = "2023-10-20T00:00:00Z"
  roles = [
    "c83a22e9-70bb-485e-a463-78a99484e5bb"
  ]
}

output "token" {
  value     = cloudtemple_iam_personal_access_token.example
  sensitive = true
}