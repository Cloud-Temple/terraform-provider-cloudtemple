---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_virtual_datacenter Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  To query this datasource you will need the compute_iaas_vmware_read role.
---

# cloudtemple_compute_virtual_datacenter (Data Source)

To query this datasource you will need the `compute_iaas_vmware_read` role.

## Example Usage

```terraform
data "cloudtemple_compute_machine_manager" "vstack-001" {
  name = "vc-vstack-001-t0001"
}

data "cloudtemple_compute_virtual_datacenter" "id" {
  id = "ac33c033-693b-4fc5-9196-26df77291dbb"
}

data "cloudtemple_compute_virtual_datacenter" "name" {
  name               = "DC-EQX6"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `machine_manager_id` (String)
- `name` (String)

### Read-Only

- `id` (String) The ID of this resource.
- `tenant_id` (String)


