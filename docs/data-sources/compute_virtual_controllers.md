---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_virtual_controllers Data Source - terraform-provider-cloudtemple"
subcategory: ""
description: |-
  
---

# cloudtemple_compute_virtual_controllers (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `virtual_machine_id` (String)

### Read-Only

- `id` (String) The ID of this resource.
- `virtual_controllers` (List of Object) (see [below for nested schema](#nestedatt--virtual_controllers))

<a id="nestedatt--virtual_controllers"></a>
### Nested Schema for `virtual_controllers`

Read-Only:

- `hot_add_remove` (Boolean)
- `id` (String)
- `label` (String)
- `summary` (String)
- `type` (String)
- `virtual_disks` (List of String)
- `virtual_machine_id` (String)

