data "cloudtemple_iam_roles" "example" {}

output "roles" {
  value = data.cloudtemple_iam_roles.example
}