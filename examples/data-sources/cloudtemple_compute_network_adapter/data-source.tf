data "cloudtemple_compute_network_adapter" "id" {
  id = "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86"
}

data "cloudtemple_compute_network_adapter" "name" {
  name               = "Network adapter 1"
  virtual_machine_id = "de2b8b80-8b90-414a-bc33-e12f61a4c05c"
}