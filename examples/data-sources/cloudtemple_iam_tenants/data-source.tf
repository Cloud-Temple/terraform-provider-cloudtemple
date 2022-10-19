data "cloudtemple_iam_tenants" "example" {}

output "tenants" {
  value = data.cloudtemple_iam_tenants.example
}