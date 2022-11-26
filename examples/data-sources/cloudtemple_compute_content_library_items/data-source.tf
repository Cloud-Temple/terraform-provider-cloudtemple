data "cloudtemple_compute_content_library" "public" {
  name = "PUBLIC"
}

data "cloudtemple_compute_content_library_items" "items" {
  content_library_id = data.cloudtemple_compute_content_library.public.id
}
