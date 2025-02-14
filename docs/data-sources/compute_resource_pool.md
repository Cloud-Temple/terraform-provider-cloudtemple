---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_resource_pool Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  To query this datasource you will need the compute_iaas_vmware_read role.
---

# cloudtemple_compute_resource_pool (Data Source)

To query this datasource you will need the `compute_iaas_vmware_read` role.

## Example Usage

```terraform
data "cloudtemple_compute_resource_pool" "id" {
  id = "d21f84fd-5063-4383-b2b0-65b9f25eac27"
}

data "cloudtemple_compute_resource_pool" "name" {
  name = "Resources"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `name` (String)

### Read-Only

- `id` (String) The ID of this resource.
- `machine_manager_id` (String)
- `metrics` (List of Object) (see [below for nested schema](#nestedatt--metrics))
- `moref` (String)
- `parent` (List of Object) (see [below for nested schema](#nestedatt--parent))

<a id="nestedatt--metrics"></a>
### Nested Schema for `metrics`

Read-Only:

- `cpu` (List of Object) (see [below for nested schema](#nestedobjatt--metrics--cpu))
- `memory` (List of Object) (see [below for nested schema](#nestedobjatt--metrics--memory))

<a id="nestedobjatt--metrics--cpu"></a>
### Nested Schema for `metrics.cpu`

Read-Only:

- `max_usage` (Number)
- `reservation_used` (Number)


<a id="nestedobjatt--metrics--memory"></a>
### Nested Schema for `metrics.memory`

Read-Only:

- `ballooned_memory` (Number)
- `max_usage` (Number)
- `reservation_used` (Number)



<a id="nestedatt--parent"></a>
### Nested Schema for `parent`

Read-Only:

- `id` (String)
- `type` (String)


