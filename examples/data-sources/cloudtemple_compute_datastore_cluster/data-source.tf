# Read an datastore cluster using its ID
data "cloudtemple_compute_datastore_cluster" "id" {
  id = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"
}

# Read an datastore cluster using its name
data "cloudtemple_compute_datastore_cluster" "name" {
  name = "sdrs001-LIVE_KOUKOU"
}