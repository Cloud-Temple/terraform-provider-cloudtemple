# Read a machine_manager using its ID
data "cloudtemple_compute_machine_manager" "id" {
  id = "9dba240e-a605-4103-bac7-5336d3ffd124"
}

# Read a machine_manager using its name
data "cloudtemple_compute_machine_manager" "name" {
  name = "vc-vstack-080-bob"
}
