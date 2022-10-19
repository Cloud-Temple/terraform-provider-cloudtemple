data "cloudtemple_iam_company" "company" {}

output "company" {
  value = data.cloudtemple_iam_company.company.name
}