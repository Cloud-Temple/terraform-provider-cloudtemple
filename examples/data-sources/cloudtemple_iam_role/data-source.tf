# Read a role using its ID
data "cloudtemple_iam_role" "id" {
  id = "c83a22e9-70bb-485e-a463-78a99484e5bb"
}

# Read a role using its name
data "cloudtemple_iam_role" "name" {
  name = "compute_read"
}