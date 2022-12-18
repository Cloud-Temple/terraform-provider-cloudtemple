## 0.4.0 (December 18, 2022)

IMPROVEMENTS:

  * The `cloudtemple_compute_virtual_machine` resource will now relocate the virtual machine when one of `virtual_datacenter_id`, `host_id`, `host_cluster_id`, `datastore_id` or `datastore_cluster_id` is changed instead of deleting and recreating it.

## 0.3.1 (December 13, 2022)

IMPROVEMENTS:

  * The `cloudtemple_compute_virtual_machine` resource now can have additional `deploy_options` specified when deploying an item of a content library.

## 0.3.0 (December 12, 2022)

NEW FEATURES:

  * The `cloudtemple_compute_content_library_item` datasource can now be used to read an item from the given content library.
  * The `cloudtemple_compute_content_library_items` datasource can now be used to read all items in a given content library.
  * The `cloudtemple_compute_virtual_machine` resource now supports deploying a new virtual machine from a content library item.

IMPROVEMENTS:

  * The provider now periodically logs information regarding the state of the activity or job running while waiting for them to complete.


## 0.2.2 (November 24, 2022)

IMPROVEMENTS:

  * Error messages returned while waiting for an activity to finish now give detailed information about the error.
  * The provider will now log HTTP requests and responses when [`TF_LOG`](https://developer.hashicorp.com/terraform/cli/config/environment-variables) is set to `DEBUG` or higher.

## 0.2.1 (November 24, 2022)

BUG FIXES:

  * The `triggered_alarms` attribute of the `cloudtemple_compute_virtual_machine` resource is now a list of objects with `id` and `status` attributes.
  * The `triggered_alarms` attribute of the `cloudtemple_compute_virtual_machine` and `cloudtemple_compute_virtual_machines` datasources is now a list of objects with `id` and `status` attributes.

IMPROVEMENTS:

  * The Go client used by the Terraform provider now automatically renew the API token before expiration.


## 0.2.0 (November 18, 2022)

BUG FIXES:

  * The arguments `address` and `scheme` in the provider configuration are now used properly.

NEW FEATURES:

  * The `cloudtemple_compute_virtual_machine` resource can now clone an already existing virtual machine using the `clone_virtual_machine_id` argument.
  * The `cloudtemple_backup_sla_policy_assignment` resource can now be used to associate SLA policies to a virtual machine.


## 0.1.0 (November 17, 2022)

NEW FEATURES:

  * The following resources are now available:
    - `cloudtemple_compute_network_adapter`
    - `cloudtemple_compute_virtual_disk`
    - `cloudtemple_compute_virtual_machine`
    - `cloudtemple_iam_personal_access_token`

 * The following data-sources are now available:
    - `cloudtemple_backup_job_sessions`
    - `cloudtemple_backup_job`
    - `cloudtemple_backup_jobs`
    - `cloudtemple_backup_metrics`
    - `cloudtemple_backup_sites`
    - `cloudtemple_backup_sla_policies`
    - `cloudtemple_backup_sla_policy`
    - `cloudtemple_backup_spp_server`
    - `cloudtemple_backup_spp_servers`
    - `cloudtemple_backup_storages`
    - `cloudtemple_backup_vcenters`
    - `cloudtemple_compute_content_libraries`
    - `cloudtemple_compute_content_library`
    - `cloudtemple_compute_datastore_cluster`
    - `cloudtemple_compute_datastore_clusters`
    - `cloudtemple_compute_datastore`
    - `cloudtemple_compute_datastores`
    - `cloudtemple_compute_folder`
    - `cloudtemple_compute_folders`
    - `cloudtemple_compute_guest_operating_system`
    - `cloudtemple_compute_guest_operating_systems`
    - `cloudtemple_compute_host_cluster`
    - `cloudtemple_compute_host_clusters`
    - `cloudtemple_compute_host`
    - `cloudtemple_compute_hosts`
    - `cloudtemple_compute_network_adapter`
    - `cloudtemple_compute_network_adapters`
    - `cloudtemple_compute_network`
    - `cloudtemple_compute_networks`
    - `cloudtemple_compute_resource_pool`
    - `cloudtemple_compute_resource_pools`
    - `cloudtemple_compute_snapshots`
    - `cloudtemple_compute_virtual_controllers`
    - `cloudtemple_compute_virtual_datacenter`
    - `cloudtemple_compute_virtual_datacenters`
    - `cloudtemple_compute_virtual_disk`
    - `cloudtemple_compute_virtual_disks`
    - `cloudtemple_compute_virtual_machine`
    - `cloudtemple_compute_virtual_machines`
    - `cloudtemple_compute_virtual_switch`
    - `cloudtemple_compute_virtual_switchs`
    - `cloudtemple_compute_worker`
    - `cloudtemple_compute_workers`
    - `cloudtemple_iam_company`
    - `cloudtemple_iam_features`
    - `cloudtemple_iam_personal_access_token`
    - `cloudtemple_iam_personal_access_tokens`
    - `cloudtemple_iam_role`
    - `cloudtemple_iam_roles`
    - `cloudtemple_iam_tenants`
    - `cloudtemple_iam_user`
    - `cloudtemple_iam_users`
