---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_content_libraries Data Source - terraform-provider-cloudtemple"
subcategory: ""
description: |-
  
---

# cloudtemple_compute_content_libraries (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `content_libraries` (List of Object) (see [below for nested schema](#nestedatt--content_libraries))
- `id` (String) The ID of this resource.

<a id="nestedatt--content_libraries"></a>
### Nested Schema for `content_libraries`

Read-Only:

- `datastore` (List of Object) (see [below for nested schema](#nestedobjatt--content_libraries--datastore))
- `id` (String)
- `machine_manager_id` (String)
- `name` (String)
- `type` (String)

<a id="nestedobjatt--content_libraries--datastore"></a>
### Nested Schema for `content_libraries.datastore`

Read-Only:

- `id` (String)
- `name` (String)

