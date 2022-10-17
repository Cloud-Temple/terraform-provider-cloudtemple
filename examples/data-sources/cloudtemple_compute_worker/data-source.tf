# Read a worker using its ID
data "cloudtemple_compute_worker" "id" {
  id = "9dba240e-a605-4103-bac7-5336d3ffd124"
}

# Read a worker using its name
data "cloudtemple_compute_worker" "name" {
  name = "vc-vstack-080-bob"
}
