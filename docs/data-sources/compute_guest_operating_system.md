---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_guest_operating_system Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  To query this datasource you will need the compute_iaas_vmware_read role.
---

# cloudtemple_compute_guest_operating_system (Data Source)

To query this datasource you will need the `compute_iaas_vmware_read` role.

## Example Usage

```terraform
data "cloudtemple_compute_guest_operating_system" "foo" {
  moref              = "amazonlinux2_64Guest"
  machine_manager_id = "9dba240e-a605-4103-bac7-5336d3ffd124"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `machine_manager_id` (String)
- `moref` (String)

### Read-Only

- `family` (String)
- `full_name` (String)
- `id` (String) The ID of this resource.


