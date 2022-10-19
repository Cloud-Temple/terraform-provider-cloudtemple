data "cloudtemple_iam_personal_access_token" "example" {
  client_id = "a1f8c60d-1441-454e-a2c0-73e4be02537e"
}

output "tokens" {
  value = data.cloudtemple_iam_personal_access_token.example
}