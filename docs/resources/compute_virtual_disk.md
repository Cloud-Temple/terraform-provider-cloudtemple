---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_virtual_disk Resource - terraform-provider-cloudtemple"
subcategory: ""
description: |-
  To manage this resource you will need the following roles:
    - compute_write
    - compute_read
    - activity_read
---

# cloudtemple_compute_virtual_disk (Resource)

To manage this resource you will need the following roles:
  - `compute_write`
  - `compute_read`
  - `activity_read`



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `capacity` (Number)
- `disk_mode` (String)
- `provisioning_type` (String)
- `virtual_machine_id` (String)

### Optional

- `controller_id` (String)
- `datastore_cluster_id` (String)
- `datastore_id` (String)

### Read-Only

- `controller_bus_number` (Number)
- `datastore_name` (String)
- `disk_path` (String)
- `disk_unit_number` (Number)
- `editable` (Boolean)
- `id` (String) The ID of this resource.
- `instant_access` (Boolean)
- `machine_manager_id` (String)
- `name` (String)
- `native_id` (String)

