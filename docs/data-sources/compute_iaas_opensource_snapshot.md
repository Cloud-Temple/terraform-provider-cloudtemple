---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_iaas_opensource_snapshot Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  Used to retrieve a specific snapshot from an Open IaaS infrastructure.
  To query this datasource you will need the compute_iaas_opensource_read role.
---

# cloudtemple_compute_iaas_opensource_snapshot (Data Source)

Used to retrieve a specific snapshot from an Open IaaS infrastructure.

To query this datasource you will need the `compute_iaas_opensource_read` role.



<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `name` (String)
- `virtual_machine_id` (String)

### Read-Only

- `create_time` (Number)
- `description` (String)
- `id` (String) The ID of this resource.

