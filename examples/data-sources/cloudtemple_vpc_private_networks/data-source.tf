data "cloudtemple_vpc_private_networks" "all" {
}

data "cloudtemple_vpc_private_networks" "filtered_by_vpc" {
  vpc_id = "960fb87a-0e84-4e6e-a6e6-d688dfefe6a8"
}
