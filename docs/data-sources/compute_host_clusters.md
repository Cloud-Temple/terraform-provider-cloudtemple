---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_host_clusters Data Source - terraform-provider-cloudtemple"
subcategory: ""
description: |-
  
---

# cloudtemple_compute_host_clusters (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `host_clusters` (List of Object) (see [below for nested schema](#nestedatt--host_clusters))
- `id` (String) The ID of this resource.

<a id="nestedatt--host_clusters"></a>
### Nested Schema for `host_clusters`

Read-Only:

- `hosts` (List of Object) (see [below for nested schema](#nestedobjatt--host_clusters--hosts))
- `id` (String)
- `machine_manager_id` (String)
- `metrics` (List of Object) (see [below for nested schema](#nestedobjatt--host_clusters--metrics))
- `moref` (String)
- `name` (String)
- `virtual_machines_number` (Number)

<a id="nestedobjatt--host_clusters--hosts"></a>
### Nested Schema for `host_clusters.hosts`

Read-Only:

- `id` (String)
- `type` (String)


<a id="nestedobjatt--host_clusters--metrics"></a>
### Nested Schema for `host_clusters.metrics`

Read-Only:

- `cpu_used` (Number)
- `memory_used` (Number)
- `storage_used` (Number)
- `total_cpu` (Number)
- `total_memory` (Number)
- `total_storage` (Number)

