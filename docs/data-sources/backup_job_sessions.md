---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_backup_job_sessions Data Source - terraform-provider-cloudtemple"
subcategory: ""
description: |-
  To query this datasource you will need the backup_read role.
---

# cloudtemple_backup_job_sessions (Data Source)

To query this datasource you will need the `backup_read` role.



<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `id` (String) The ID of this resource.
- `job_sessions` (List of Object) (see [below for nested schema](#nestedatt--job_sessions))

<a id="nestedatt--job_sessions"></a>
### Nested Schema for `job_sessions`

Read-Only:

- `duration` (Number)
- `end` (Number)
- `id` (String)
- `job_id` (String)
- `job_name` (String)
- `sla_policies` (List of Object) (see [below for nested schema](#nestedobjatt--job_sessions--sla_policies))
- `sla_policy_type` (String)
- `start` (Number)
- `statistics` (List of Object) (see [below for nested schema](#nestedobjatt--job_sessions--statistics))
- `status` (String)
- `type` (String)

<a id="nestedobjatt--job_sessions--sla_policies"></a>
### Nested Schema for `job_sessions.sla_policies`

Read-Only:

- `href` (String)
- `id` (String)
- `name` (String)


<a id="nestedobjatt--job_sessions--statistics"></a>
### Nested Schema for `job_sessions.statistics`

Read-Only:

- `failed` (Number)
- `skipped` (Number)
- `success` (Number)
- `total` (Number)

