data "cloudtemple_compute_content_library" "public" {
  name = "PUBLIC"
}

# Read a content library item using its ID
data "cloudtemple_compute_content_library_item" "id" {
  content_library_id = data.cloudtemple_compute_content_library.public.id
  id                 = "8faded09-9f8b-4e27-a978-768f72f8e5f8"
}

# Read a content library item using its name
data "cloudtemple_compute_content_library_item" "name" {
  content_library_id = data.cloudtemple_compute_content_library.public.id
  name               = "20211115132417_master_linux-centos-8"
}