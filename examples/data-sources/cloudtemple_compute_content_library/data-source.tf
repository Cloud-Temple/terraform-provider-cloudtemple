# Read a content library using its ID
data "cloudtemple_compute_content_library" "id" {
  id = "355b654d-6ea2-4773-80ee-246d3f56964f"
}

# Read a content library using its name
data "cloudtemple_compute_content_library" "name" {
  name = "PUBLIC"
}