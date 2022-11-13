---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_backup_vcenters Data Source - terraform-provider-cloudtemple"
subcategory: ""
description: |-
  To query this datasource you will need the backup_read role.
---

# cloudtemple_backup_vcenters (Data Source)

To query this datasource you will need the `backup_read` role.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `spp_server_id` (String)

### Read-Only

- `id` (String) The ID of this resource.
- `vcenters` (List of Object) (see [below for nested schema](#nestedatt--vcenters))

<a id="nestedatt--vcenters"></a>
### Nested Schema for `vcenters`

Read-Only:

- `id` (String)
- `instance_id` (String)
- `internal_id` (Number)
- `name` (String)
- `spp_server_id` (String)

