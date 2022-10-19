data "cloudtemple_iam_users" "example" {}

output "users" {
  value = data.cloudtemple_iam_users.example.users
}