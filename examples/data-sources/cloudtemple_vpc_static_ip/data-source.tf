# Read a static IP using its ID
data "cloudtemple_vpc_static_ip" "by_id" {
  id = "4f759498-05ff-42ec-b40e-90c1c9c77541"
}

# Read a static IP using the MAC address of its network adapter
data "cloudtemple_vpc_static_ip" "by_mac" {
  mac_address = "00:50:56:ab:cd:ef"
}
