data "cloudtemple_iam_role" "example" {
  id = "b7a0d310-839e-419b-9e56-fb052eb17958"
}

output "role" {
  value = data.cloudtemple_iam_role.example
}