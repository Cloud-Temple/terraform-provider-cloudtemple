data "cloudtemple_iam_features" "example" {}

output "features" {
  value = data.cloudtemple_iam_features.example
}