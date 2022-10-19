data "cloudtemple_iam_role" "example" {}

output "role" {
  value = data.cloudtemple_iam_role.example
}