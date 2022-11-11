---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_guest_operating_systems Data Source - terraform-provider-cloudtemple"
subcategory: ""
description: |-
  
---

# cloudtemple_compute_guest_operating_systems (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `machine_manager_id` (String)

### Read-Only

- `guest_operating_systems` (List of Object) (see [below for nested schema](#nestedatt--guest_operating_systems))
- `id` (String) The ID of this resource.

<a id="nestedatt--guest_operating_systems"></a>
### Nested Schema for `guest_operating_systems`

Read-Only:

- `family` (String)
- `full_name` (String)
- `moref` (String)

