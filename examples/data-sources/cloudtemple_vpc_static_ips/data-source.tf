# Read all static IPs of a private network
data "cloudtemple_vpc_static_ips" "all" {
  private_network_id = "c229c411-ac30-4caa-9c67-70d4c230d0ee"
}

# Read the static IPs of a private network for a specific virtual machine
data "cloudtemple_vpc_static_ips" "by_vm" {
  private_network_id = "c229c411-ac30-4caa-9c67-70d4c230d0ee"
  virtual_machine_id = "fc3a1ed8-737f-4667-acaa-320d3f523b6f"
}
