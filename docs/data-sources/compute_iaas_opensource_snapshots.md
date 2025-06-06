---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_iaas_opensource_snapshots Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  Used to retrieve all snapshots from an Open IaaS infrastructure for a specific virtual machine.
  To query this datasource you will need the compute_iaas_opensource_read role.
---

# cloudtemple_compute_iaas_opensource_snapshots (Data Source)

Used to retrieve all snapshots from an Open IaaS infrastructure for a specific virtual machine.

To query this datasource you will need the `compute_iaas_opensource_read` role.

## Example Usage

```terraform
// Data source to retrieve information about a list of snapshots in CloudTemple IaaS OpenSource.

# This example demonstrates how to use the data source to get all the snapshots of an Open IaaS infrastructure.
data "cloudtemple_compute_iaas_opensource_snapshots" "all-snapshots" {}

output "snapshot" {
  value = data.cloudtemple_compute_iaas_opensource_snapshots.all-snapshots
}

# This example demonstrates how to use the data source to get all the snapshots of a virtual machine.
data "cloudtemple_compute_iaas_opensource_snapshots" "all-snapshots-of-vm" {
  virtual_machine_id = "virtual_machine_id"
}

output "snapshot" {
  value = data.cloudtemple_compute_iaas_opensource_snapshots.all-snapshots-of-vm
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `virtual_machine_id` (String) The ID of the virtual machine to retrieve snapshots for.

### Read-Only

- `id` (String) The ID of this resource.
- `snapshots` (List of Object) List of snapshots for the specified virtual machine. (see [below for nested schema](#nestedatt--snapshots))

<a id="nestedatt--snapshots"></a>
### Nested Schema for `snapshots`

Read-Only:

- `create_time` (Number)
- `description` (String)
- `id` (String)
- `name` (String)
- `virtual_machine_id` (String)


