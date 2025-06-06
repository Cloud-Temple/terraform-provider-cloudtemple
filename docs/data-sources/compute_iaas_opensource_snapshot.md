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

## Example Usage

```terraform
// Data source to retrieve information about a snapshot in CloudTemple IaaS OpenSource.

# This example demonstrates how to use the data source to get a snapshot by its ID.
data "cloudtemple_compute_iaas_opensource_snapshot" "snapshot-1" {
  id = "snapshot_id"
}

output "snapshot" {
  value = data.cloudtemple_compute_iaas_opensource_snapshot.snapshot-1
}

# This example demonstrates how to use the data source to get a snapshot by its name and virtual machine ID.
data "cloudtemple_compute_iaas_opensource_snapshot" "snapshot-2" {
  name               = "snapshot_name"
  virtual_machine_id = "virtual_machine_id"
}

output "snapshot" {
  value = data.cloudtemple_compute_iaas_opensource_snapshot.snapshot-2
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `id` (String) The ID of the snapshot to retrieve. Conflicts with `name`.
- `name` (String) The name of the snapshot to retrieve. Conflicts with `id`.
- `virtual_machine_id` (String) The ID of the virtual machine the snapshot belongs to. Required when searching by `name`.

### Read-Only

- `create_time` (Number) The timestamp when the snapshot was created.
- `description` (String) The description of the snapshot.


