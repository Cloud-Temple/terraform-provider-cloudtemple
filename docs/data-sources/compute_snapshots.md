---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_snapshots Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  To query this datasource you will need the compute_iaas_vmware_read role.
---

# cloudtemple_compute_snapshots (Data Source)

To query this datasource you will need the `compute_iaas_vmware_read` role.

## Example Usage

```terraform
data "cloudtemple_compute_snapshots" "foo" {
  virtual_machine_id = "de2b8b80-8b90-414a-bc33-e12f61a4c05c"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `virtual_machine_id` (String)

### Read-Only

- `id` (String) The ID of this resource.
- `snapshots` (List of Object) (see [below for nested schema](#nestedatt--snapshots))

<a id="nestedatt--snapshots"></a>
### Nested Schema for `snapshots`

Read-Only:

- `create_time` (Number)
- `id` (String)
- `name` (String)
- `virtual_machine_id` (String)


