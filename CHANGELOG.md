***Warning: Using "Release Candidate" versions (-rc.X) in a **production environment** is **strongly discouraged**, as they may contain unresolved bugs and pose risks to the stability and security of your systems.***

## 1.5.2 (Not released yet.)
<img id="latest" src="https://badgen.net/badge/channel/latest/yellow" alt="Channel: latest" />

BUG FIXES :

  * Fixed a bug causing `cloudtemple_compute_iaas_opensource_network_adapter` creation to fail with "Must be a MAC address" error when MAC address is not explicitly specified.

## 1.5.1 (November 25th, 2025)

BUG FIXES :

  * Fixed a bug causing `cloudtemple_compute_iaas_opensource_virtual_machine` creation to fail due to missing `disabled` option in `high_availability' property.

## 1.5.0 (November 18th, 2025)

NEW FEATURES :
  * Added resource `cloudtemple_object_storage_bucket`.
  * Added resource `cloudtemple_object_storage_storage_account`.
  * Added resource `cloudtemple_object_storage_global_access_key`.
  * Added resource `cloudtemple_object_storage_acl_entry`.
  * Added datasource `cloudtemple_object_storage_bucket`.
  * Added datasource `cloudtemple_object_storage_buckets`.
  * Added datasource `cloudtemple_object_storage_bucket_files`.
  * Added datasource `cloudtemple_object_storage_storage_account`.
  * Added datasource `cloudtemple_object_storage_storage_accounts`.
  * Added datasource `cloudtemple_object_storage_role`.
  * Added datasource `cloudtemple_object_storage_roles`.
  * Added datasource `cloudtemple_object_storage_acl`.

## 1.4.0 (October 30th, 2025)

NEW FEATURES :
  * Added datasource `cloudtemple_marketplace_item` used to retreive Virtual Machine Images (VMI) from the new console's marketplace.
  * Added property `marketplace_item_id` in resource `cloudtemple_compute_virtual_machine` used to deploy a virtual machine from a VMI of the Marketplace.
  * Added property `marketplace_item_id` in resource `cloudtemple_compute_iaas_opensource_virtual_machine` used to deploy a virtual machine from a VMI of the Marketplace.

## 1.3.0 (October 30th, 2025)
<img id="stable" src="https://badgen.net/badge/channel/stable/green" alt="Channel: stable" />

INFORMATIONS :

  * Promoted `v1.3.0-rc.1` from latest build to stable release. No functional changes.

## 1.3.0-rc.1 (October 9th, 2025)

IMPROVMENTS :

  * Added new resource `cloudtemple_compute_iaas_opensource_replication_policy`.
  * Added new datasource `cloudtemple_compute_iaas_opensource_replication_policy`.
  * Added new datasource `cloudtemple_compute_iaas_opensource_replication_policies`.
  * Added new property `replication_policy_id` on resource `cloudtemple_compute_iaas_opensource_virtual_machine`.

## 1.2.0-rc.1 (October 2nd, 2025)  

IMPROVEMENTS :

  * Added `os_disk` and `os_network_adapter` configuration blocks on resource `cloudtemple_compute_iaas_opensource_virtual_machine`. (Use those to reference the disks and network adapters that are emedded in the template used).
  * Improved resource `cloudtemple_compute_iaas_opensource_virtual_disk` so that it can be update in-place rather than being recreated. It can now be updated, resized, relocated, attached and detached.
  * Added properties `guestinfo.userdata`, `guestinfo.userdata.encoding`, `guestinfo.metadata`, `guestinfo.metadata.encoding` to the available extra_config keys in the resource `cloudtemple_compute_virtual_machine` resource.
  * Added property `tx_checksumming` to the resource `cloudtemple_compute_iaas_opensource_network_adapter`.

## 1.1.0-rc.1 (September 9th, 2025)

IMPROVEMENTS :

  * Added automatic state migration for `extra_config` in resource `cloudtemple_compute_virtual_machine`.
    - The `extra_config` parameter format has been changed from array of objects to a map for better usability
    - Existing Terraform states with the old format will be automatically migrated to the new format
    - Old format: `[{"key": "svga.present", "value": "TRUE"}]`
    - New format: `{"svga.present": "TRUE"}`
    - This migration is transparent and requires no user action

NEW FEATURES :

  * Added update extra config to ressource `cloudtemple_compute_virtual_machine`.
    - Supported keys: `guestinfo.ignition.config.data`, `guestinfo.ignition.config.data.encoding`, `guestinfo.afterburn.initrd.network-kargs`, `stealclock.enable`, `disk.enableUUID`, `pciPassthru.use64BitMMIO`, `pciPassthru.64bitMMioSizeGB`

## 1.0.1 (September 8th, 2025)

BUG FIXES:

  * Fixed a bug causing fails while deploying from the content library, when `power_on` parameter is set to `true`.

## 1.0.0 (September 5th, 2025)

BUG FIXES:

  * Fixed a bug causing resources `cloudtemple_compute_virtual_disk` to be imported as `os_disk` in the state of the attached `cloudtemple_compute_virtual_machine`.
  * Fixed a bug in the datasources `cloudtemple_compute_iaas_opensource_backup_policy` and `cloudtemple_compute_iaas_opensource_backup_policies`.

IMPROVEMENTS:

  * Added memoryReservation feature for virtual machines
  * Configuration parameters `os_disk` and `os_network_adapter` are now imported with the resource `cloudtemple_compute_virtual_machine` (Experimentation)
  * Standardized state management code across all datasources and resources [#155]
    - Implemented consistent data mapping patterns for all components
    - Added standardized error handling and state management
    - Unified the way data is processed and stored across resources
    - Improved error handling in API client
  * Improved provider documentation
    - Added detailed examples for all data sources and resources
    - Enhanced documentation clarity and completeness
    - Updated all resource and data source documentation with consistent formatting
    - Added new examples for iaas_opensource components
  * Refactored acceptance tests for better reliability and maintainability
    - Moved all test files to dedicated test directories
    - Standardized test patterns across all components
    - Added test coverage for iaas_opensource features
  * Added new helper functions for each resource type to improve code reusability
    - Created 40+ new helper files for better code organization
    - Implemented shared functionality for common operations
    - Reduced code duplication across similar resources

BREAKING CHANGES:

  * Removed property `mac_type` from resource `cloudtemple_compute_network_adapter`. 
    - MAC addresses are now generated whenever the `mac_address` property is not explicitly provided by the user.
    - Same behavior is now applied to the `os_network_adapter` block of the resource `cloudtemple_compute_virtual_machine`.

CODE ORGANIZATION:

  * Major structural improvements:
    - Moved all test files into dedicated test folders:
      * internal/client/tests/ for client tests
      * internal/provider/tests/ for provider tests
    - Created separate helper files for each resource type in internal/provider/helpers/
    - Standardized file organization across the codebase
  * New helper files added:
    - Added helper files for backup components
    - Added helper files for compute components
    - Added helper files for IAM components
  * Improved code modularity:
    - Separated data mapping logic into dedicated helper files
    - Centralized common functionality in helper packages
    - Standardized resource and data source implementations
    - Improved API client organization
  * Standardized data mapping patterns across all components:
    - All singular datasources now follow consistent patterns
    - Implemented standardized data mapping for iaas_opensource components
    - Added flatten pattern for nested properties
    - Unified error handling and state management across all resources

## 1.0.0-rc.3 (May 15th, 2025)

BUG FIXES:

  * Fixed a bug causing resources `cloudtemple_compute_virtual_disk` to be imported as `os_disk` in the state of the attached `cloudtemple_compute_virtual_machine`.

## 1.0.0-rc.2 (May 6th, 2025)

BUG FIXES:

  * Fixed a bug in the datasources `cloudtemple_compute_iaas_opensource_backup_policy` and `cloudtemple_compute_iaas_opensource_backup_policies`.

## 1.0.0-rc.1 (May 5th, 2025)

IMPROVEMENTS:

  * Added memoryReservation feature for virtual machines
  * Configuration parameters `os_disk` and `os_network_adapter` are now imported with the resource `cloudtemple_compute_virtual_machine` (Experimentation)
  * Standardized state management code across all datasources and resources [#155]
    - Implemented consistent data mapping patterns for all components
    - Added standardized error handling and state management
    - Unified the way data is processed and stored across resources
    - Improved error handling in API client
  * Improved provider documentation
    - Added detailed examples for all data sources and resources
    - Enhanced documentation clarity and completeness
    - Updated all resource and data source documentation with consistent formatting
    - Added new examples for iaas_opensource components
  * Refactored acceptance tests for better reliability and maintainability
    - Moved all test files to dedicated test directories
    - Standardized test patterns across all components
    - Added test coverage for iaas_opensource features
  * Added new helper functions for each resource type to improve code reusability
    - Created 40+ new helper files for better code organization
    - Implemented shared functionality for common operations
    - Reduced code duplication across similar resources

BREAKING CHANGES:

  * Removed property `mac_type` from resource `cloudtemple_compute_network_adapter`. 
    - MAC addresses are now generated whenever the `mac_address` property is not explicitly provided by the user.
    - Same behavior is now applied to the `os_network_adapter` block of the resource `cloudtemple_compute_virtual_machine`.

CODE ORGANIZATION:

  * Major structural improvements:
    - Moved all test files into dedicated test folders:
      * internal/client/tests/ for client tests
      * internal/provider/tests/ for provider tests
    - Created separate helper files for each resource type in internal/provider/helpers/
    - Standardized file organization across the codebase
  * New helper files added:
    - Added helper files for backup components
    - Added helper files for compute components
    - Added helper files for IAM components
  * Improved code modularity:
    - Separated data mapping logic into dedicated helper files
    - Centralized common functionality in helper packages
    - Standardized resource and data source implementations
    - Improved API client organization
  * Standardized data mapping patterns across all components:
    - All singular datasources now follow consistent patterns
    - Implemented standardized data mapping for iaas_opensource components
    - Added flatten pattern for nested properties
    - Unified error handling and state management across all resources

## 0.16.3 (April 4th, 2025)

BUG FIXES :

  * Fixed a bug causing the provider to wait indefinitely the result of an activity when the PAT doesn't have the `activity_read` permission. 

## 0.16.3-rc.1 (April 3rd, 2025)

MISCELLANEOUS :

  * Updated the way the ID of the `cloudtemple_compute_iaas_opensource_virtual_machine` resource is retreived from the activity.

## 0.16.2 (April 2nd, 2025)

NEW FEATURES :

  * Added import capability to resources:
    - `cloudtemple_COMPUTE_IAAS_OPENSOURCE_network_adapter`
    - `cloudtemple_iam_personal_access_token`
  
IMPROVEMENTS :

  * Added import documentation to resources:
    - `cloudtemple_backup_sla_policy_assignment`
    - `cloudtemple_compute_iaas_opensource_virtual_disk`
    - `cloudtemple_compute_iaas_opensource_virtual_machine`
    - `cloudtemple_compute_virtual_controller`
    - `cloudtemple_compute_virtual_disk`

BUG FIXES :

  * Fixed provider documentation to properly display client_id and secret_id as required fields by customizing the template

## 0.16.1 (February 27th, 2025)

NEW FEATURES :

  * Added import capability to resource `cloudtemple_compute_iaas_opensource_virtual_disk`.

IMPROVEMENTS :

  * Added automatic power cycling for `cloudtemple_compute_iaas_opensource_virtual_machine` when CPU, memory, or cores per socket are updated on a running VM.

BUG FIXES :

  * Fixed a bug in `cloudtemple_compute_iaas_opensource_virtual_disk` resource that prevented Terraform from recreating the resource when it was deleted outside of Terraform by properly setting the ID to empty when the resource is not found.

## 0.16.0 (February 18th, 2025)

NEW FEATURES :

  * Added cloud-init to resource `cloudtemple_compute_iaas_opensource_virtual_machine`.
  * Added ability to update the boot firmware of a `cloudtemple_compute_iaas_opensource_virtual_machine`.

BUG FIXES :

  * Fixed a bug causing communications with the Backup module to crash.

## 0.15.2 (January 24th, 2025)

MISCELLANEOUS :

  * Added examples and documentation about the new Open IaaS features.
  * Improved error handling of new Open IaaS datasources.

## 0.15.1 (January 23nd, 2025)

BUG FIXES :

  * Fixed a bug causing provider plugin to crash when using a datasource from iaas opensource with an ID.

NEW FEATURES :

  * Added ability to mount and unmount ISO files on `cloudtemple_compute_iaas_opensource_virtual_machine`.

## 0.15.0 (December 20th, 2024)

NEW FEATURES :

  * The following resources are now available :
    - `cloudtemple_compute_iaas_opensource_virtual_machine`
    - `cloudtemple_compute_iaas_opensource_virtual_disk`
    - `cloudtemple_compute_iaas_opensource_network_adapter`

  * The following datasource are now available :
    - `cloudtemple_compute_iaas_opensource_machine_manager`
    - `cloudtemple_compute_iaas_opensource_pool`
    - `cloudtemple_compute_iaas_opensource_host`
    - `cloudtemple_compute_iaas_opensource_network`
    - `cloudtemple_compute_iaas_opensource_storage_repository`
    - `cloudtemple_compute_iaas_opensource_template`
    - `cloudtemple_compute_iaas_opensource_virtual_machine`
    - `cloudtemple_compute_iaas_opensource_virtual_disk`
    - `cloudtemple_compute_iaas_opensource_snapshot`
    - `cloudtemple_compute_iaas_opensource_network_adapter`
    - `cloudtemple_backup_iaas_opensource_backup`
    - `cloudtemple_backup_iaas_opensource_policy`
    
## 0.14.1 (November 28th, 2024)

BUG FIXES:

  * Fixed `cloudtemple_virtual_machines` and `cloudtemple_compute_networks` datasources that were not working properly.

## 0.14.0 (October 31st, 2024)

BUG FIXES:

  * Fixed a bug causing terraform plugin to fail when an alarm is triggered on a `cloudtemple_compute_virtual_machine`.

MISCELLANEOUS:

  * Updated the role names in the documentations.
  * Cloud-init now doesn't force recreate when updated.

## 0.13.0-rc.1 (March 19th, 2024)

NEW FEATURES:

  * Added the ability to create NVME controllers.

BUG FIXES:

  * Fixed a bug causing `datastore_id` and `datastore_cluster_id` not to be imported in state when importing a `cloudtemple_compute_virtual_machine`

## 0.12.4-rc.3 (February 13th, 2024)

BUG FIXES :

  * Fixed a bug causing tags deletion to fail the process.
  * Fixed a bug causing `cloudtemple_compute_virtual_disk` creation to fail.

## 0.12.4-rc.2 (February 8th, 2024)

BUG FIXES :

  * Fixed a bug causing backup module not to find the virtual disk after running hypervisor inventory.

## 0.12.4-rc.1 (February 2nd, 2024)

BUG FIXES :

  * Fixed a bug when retreiving datastore information due to API modification.

## 0.12.3 (November 29th, 2023)

BUG FIXES :

  * Fixed a bug causing terraform to taint a healthy virtual_disk due to controller_id desynchronization in state.

## 0.12.2 (November 28th, 2023)

BUG FIXES:

  * Fixed a bug causing plugin to crash when using the datasource cloudtemple_compute_content_library_items

## 0.12.1 (November 23rd, 2023)

BUG FIXES:

  * Fixed a bug on newly released Virtual Machine Guest OS Customization feature.

## 0.12.0 (November 22nd, 2023)

NEW FEATURES:

  * Guest OS of `cloudtemple_compute_virtual_machine` can now be customized using `customize` block.

## 0.11.0 (November 15th, 2023)

NEW FEATURES:

  * Added a new parameter `expose_hardware_virtualization` on resource `cloudtemple_cirtual_machine` that enables nested hardware virtualization on the virtual machine.

MISCELLANEOUS:

  * Changed the way URL are built in the Go Http client.

BUG FIXES:

  * Fixed a bug on resource `cloudtemple_compute_virtual_disk` causing desynchronization between state and real configuration after an import.
  * Fixed a bug on resource `cloudtemple_compute_virtual_controller` causing desynchronization between state and real configuration after an import.

## 0.10.0-rc.2 (November 8th, 2023)

NEW FEATURES:

  * Resource `cloudtemple_compute_virtual_disk` can now be imported into state.

## 0.10.0-rc.1 (November 7th, 2023)

NEW FEATURES:

  * Added support of creation, management and deletion of virtual controllers through a new `cloudtemple_compute_virtual_controller` resource.

## 0.9.0-rc.1 (November 2nd, 2023)

NEW FEATURES:

  * Boot options of a virtual machine can now be modified.

## 0.8.2 (October 11th, 2023)

BUG FIXES :

  * Fixed a bug causing datasource `cloudtemple_compute_datastore_cluster` not to work.

## 0.8.1 (October 11th, 2023)

IMPROVEMENTS:

  * Added filters on data source cloudtemple_compute_virtual_switchand cloudtemple_compute_virtual_switchs
  * Added filters on data source cloudtemple_compute_networkand cloudtemple_compute_networks
  * Updated datasource cloudtemple_compute_datastore_cluster to make the filter datacenter_id mandatory

## 0.8.0 (October 4th, 2023)

NEW FEATURES:

  * Added a property `disks_provisioning_type` that overrides the provisioning types of disks present on a OVF deployed from content library.

IMPROVEMENTS:
  
  * Added `name` and `machine_manager` filters on datasource `cloudtemple_compute_content_library` and `cloudtemple_compute_content_libraries`
  * Added `name` and `content_library_id` filters on datasource `cloudtemple_compute_content_library_item` and `cloudtemple_compute_content_library_items`

BUG FIXES:

  * Updated the CreateContext of resource cloudtemple_compute_virtual_machine to make it without timeout so it doesn't fails after 20 minutes.
  * Fixed a bug causing backup module not to find the virtual machine when trying to assign an SLA policy

## 0.7.0 (September 22, 2023)

NEW FEATURES:

  * Added cloud-init support

## 0.6.1 (June 19, 2023)

BUG FIXES:

  * Fixed mistyped property on newly added 'backup_virtual_machine' controller causes virtual machines struggling to power on.

## 0.6.0 (June 16, 2023)

BUG FIXES:

  * Fixed a bug causing crashes when trying to start a `cloudtemple_compute_virtual_machine` because it has pending recommendation(s) on VMWare side.
  * Fixed wrong backup job running after `cloudtemple_compute_virtual_machine` create or update.
  * Fixed a bug causing fails when trying to create a resource `cloudtemple_compute_virtual_machine`.
  * Fixed tfstate incorrectly refreshing when updating property `backup_sla_policies` of `cloudtemple_compute_virtual_machine` from outside the provider.
  * Fixed empty recommendations causing `cloudtemple_compute_virtual_machine` not starting up.
  * Virtual machines are now inventoried by the backup server when they are created from clone or content library.
  * Fixed a bug preventing resource `cloudtemple_compute_virtual_machine` to power on when created from the CL or a Clone.
  * Fixed a bug causing preventing resource `cloudtemple_compute_virtual_machine` to be inventoried when property `backup_sla_policies` is set after creation.

IMPROVEMENTS:

  * Added property `backup_sla_policies` to resource `cloudtemple_compute_virtual_machine`, so that it can be created and started in an SNC environment.
  * `datastore_cluster_id` and `datastore_id` now conflicts each other on resource `cloudtemple_compute_virtual_disk` and at least one of them is now required.
  * Resource `cloudtemple_compute_network_adapter` is now importable.
  * Property `guest_operating_system_moref` on resource `cloudtemple_compute_virtual_machine` can now be updated.
  * Resource `cloudtemple_compute_virtual_machine` is now powered off before delete.
  * Added missing documentation on import of resource `cloudtemple_compute_virtual_machine`.
  * Implemented filters on following data_sources :
    - `cloudtemple_compute_datastore`
    - `cloudtemple_compute_datastore_cluster`
    - `cloudtemple_compute_host_cluster`
    - `cloudtemple_compute_datacenter`
    - `datastores`
    - `datastore_clusters`
    - `host_clusters`
    - `virtual_machines`
  * Implemented importation of os_disk and os_network_adapter in the resource `cloudtemple_compute_virtual_machine` when deployed from a Content Library or a Clone.
  * Added property `backup_sla_policies` on resource `cloudtemple_compute_virtual_disk`.
  * Property `guest_operating_system_moref` on resource `cloudtemple_compute_virtual_machine` can now be computed.
  * Property `backup_sla_policies` of `cloudtemple_compute_virtual_machine` is now optional.
  * Renamed data source `cloudtemple_compute_worker` to `cloudtemple_compute_machine_manager`.

## 0.6.0-rc.2 (May 24, 2023)

BUG FIXES:

  * Fixed tfstate incorrectly refreshing when updating property `backup_sla_policies` of `cloudtemple_compute_virtual_machine` from outside the provider.
  * Fixed empty recommendations causing `cloudtemple_compute_virtual_machine` not starting up.
  * Virtual machines are now inventoried by the backup server when they are created from clone or content library.

IMPROVEMENTS:

  * Property `guest_operating_system_moref` on resource `cloudtemple_compute_virtual_machine` can now be computed.
  * Property `backup_sla_policies` of `cloudtemple_compute_virtual_machine` is now optional.
  * Renamed data source `cloudtemple_compute_worker` to `cloudtemple_compute_machine_manager`.

## 0.6.0-rc.1 (April 25, 2023)

BUG FIXES:

  * Fixed a bug causing crashes when trying to start a `cloudtemple_compute_virtual_machine` because it has pending recommendation(s) on VMWare side.
  * Fixed wrong backup job running after `cloudtemple_compute_virtual_machine` create or update.
  * Fixed a bug causing fails when trying to create a resource `cloudtemple_compute_virtual_machine`.

IMPROVEMENTS:

  * Added property `backup_sla_policies` to resource `cloudtemple_compute_virtual_machine`, so that it can be created and started in an SNC environment.
  * `datastore_cluster_id` and `datastore_id` now conflicts each other on resource `cloudtemple_compute_virtual_disk` and at least one of them is now required.
  * Resource `cloudtemple_compute_network_adapter` is now importable.
  * Property `guest_operating_system_moref` on resource `cloudtemple_compute_virtual_machine` can now be updated.
  * Resource `cloudtemple_compute_virtual_machine` is now powered off before delete.
  * Added missing documentation on import of resource `cloudtemple_compute_virtual_machine`.
  * Implemented filters on following data_sources :
    - `cloudtemple_compute_datastore`
    - `cloudtemple_compute_datastore_cluster`
    - `cloudtemple_compute_host_cluster`
    - `cloudtemple_compute_datacenter`

## 0.5.0 (March 16, 2023)

BUG FIXES:

  * The `datacenter_id` replaces the `virtual_datacenter_id` argument in the `compute_virtual_machine` resource. `virtual_datacenter_id` was deprecated and has been removed.

  * The `datacenter_id` replaces the `virtual_datacenter_id` attribute in the `compute_virtual_machine` and `compute_virtual_machines` datasources. `virtual_datacenter_id` was deprecated and has been removed.

## 0.4.2 (February 3, 2023)

BUG FIXES:

  * Fixed a panic occuring in `cloudtemple_compute_virtual_machine` when an error happened while reading a virtual machine information.

## 0.4.1 (December 22, 2022)

BUG FIXES:

  * The `cloudtemple_compute_network_adapter` resource will now clean up broken network adapters when an error occurs while creating it.
  * The `cloudtemple_compute_virtual_disk` resource will now clean up broken virtual disks when an error occurs while creating it.
  * The `cloudtemple_compute_virtual_machine` resource will now clean up broken virtual machines when an error occurs while creating it.

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
