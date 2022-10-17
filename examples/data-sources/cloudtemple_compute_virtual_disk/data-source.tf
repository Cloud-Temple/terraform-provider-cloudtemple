# Read a virtual disk using its ID
data "cloudtemple_compute_virtual_disk" "id" {
  id = "d370b8cd-83eb-4315-a5d9-42157e2e4bb4"
}

# Using a virtual disk using its name
data "cloudtemple_compute_virtual_disk" "name" {
  name               = "Hard disk 1"
  virtual_machine_id = "de2b8b80-8b90-414a-bc33-e12f61a4c05c"
}