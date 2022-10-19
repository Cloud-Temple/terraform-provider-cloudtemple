data "cloudtemple_iam_user" "remi" {
  id = "37105598-4889-43da-82ea-cf60f2a36aee"
}

output "remi" {
  value = data.cloudtemple_iam_user.remi
}