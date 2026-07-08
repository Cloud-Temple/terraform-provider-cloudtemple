# Read all private networks
data "cloudtemple_vpc_private_networks" "all" {}

# Read the private networks of a specific VPC
data "cloudtemple_vpc_private_networks" "by_vpc" {
  vpc_id = "aff7e62f-c603-419d-ae0c-03441abf0655"
}
