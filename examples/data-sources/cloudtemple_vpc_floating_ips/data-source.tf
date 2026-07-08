# Read all floating IPs
data "cloudtemple_vpc_floating_ips" "all" {}

# Read the floating IPs bound to a static IP in a specific VPC
data "cloudtemple_vpc_floating_ips" "by_vpc" {
  vpc_id = "39ea5bbe-50ff-49b1-82b7-a6857b9aea4c"
}
