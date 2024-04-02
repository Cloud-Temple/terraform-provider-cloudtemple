package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceVirtualDisk() *schema.Resource {
	return &schema.Resource{
		Description: "",

		CreateWithoutTimeout: computeVirtualDiskCreate,
		ReadContext:          computeVirtualDiskRead,
		UpdateContext:        computeVirtualDiskUpdate,
		DeleteContext:        computeVirtualDiskDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"controller_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"provisioning_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: `provisioning_type can take 3 different values :
					- staticImmediate
					- dynamic
					- staticDiffered`,
			},
			"disk_mode": {
				Type:     schema.TypeString,
				Required: true,
				Description: `disk_mode can have multiple different values :
					- Persistent: Changes are immediately and permanently written to the virtual disk.
					- Independent non persistent: Changes to virtual disk are made to a redo log and discarded at power off. Not affected by snapshots.
					- Independent persistent: Changes are immediately and permanently written to the virtual disk. Not affected by snapshots.`,
			},
			"capacity": {
				Type:        schema.TypeInt,
				Description: "Disk size in bytes",
				Required:    true,
			},
			"datastore_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the datastore. Conflict with `datastore_cluster_id`.",
				ForceNew:      true,
				Optional:      true,
				Computed:      true,
				AtLeastOneOf:  []string{"datastore_id", "datastore_cluster_id"},
				ConflictsWith: []string{"datastore_cluster_id"},
				ValidateFunc:  validation.IsUUID,
			},
			"datastore_cluster_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the datastore cluster. Conflict with `datastore_id`.",
				ForceNew:      true,
				Optional:      true,
				Computed:      true,
				AtLeastOneOf:  []string{"datastore_id", "datastore_cluster_id"},
				ConflictsWith: []string{"datastore_id"},
				ValidateFunc:  validation.IsUUID,
			},
			"backup_sla_policies": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The IDs of the SLA policies to assign to the virtual machine.",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
			},

			// Out
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"disk_unit_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"controller_bus_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"datastore_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"instant_access": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"native_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"disk_path": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"editable": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func computeVirtualDiskCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().VirtualDisk().Create(ctx, &client.CreateVirtualDiskRequest{
		ControllerId:       d.Get("controller_id").(string),
		ProvisioningType:   d.Get("provisioning_type").(string),
		DiskMode:           d.Get("disk_mode").(string),
		Capacity:           d.Get("capacity").(int),
		VirtualMachineId:   d.Get("virtual_machine_id").(string),
		DatastoreId:        d.Get("datastore_id").(string),
		DatastoreClusterId: d.Get("datastore_cluster_id").(string),
	})
	if err != nil {
		return diag.Errorf("the virtual disk could not be created: %s", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	setIdFromActivityConcernedItems(d, activity, "virtual_disk")
	if err != nil {
		return diag.Errorf("failed to create virtual disk, %s", err)
	}

	if len(d.Get("backup_sla_policies").(*schema.Set).List()) > 0 {
		// First we need to update the catalog
		jobs, err := c.Backup().Job().List(ctx, &client.BackupJobFilter{
			Type: "catalog",
		})
		if err != nil {
			return diag.Errorf("failed to find catalog job: %s", err)
		}

		var job = &client.BackupJob{}
		for _, currJob := range jobs {
			if currJob.Name == "Hypervisor Inventory" {
				job = currJob
			}
		}

		activityId, err := c.Backup().Job().Run(ctx, &client.BackupJobRunRequest{
			JobId: job.ID,
		})
		if err != nil {
			return diag.Errorf("failed to update catalog: %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to update catalog, %s", err)
		}

		_, err = c.Backup().VirtualDisk().WaitForInventory(ctx, d.Id(), getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to find virtual disk in backup inventory : %s", err)
		}

		slaPolicies := []string{}
		for _, policy := range d.Get("backup_sla_policies").(*schema.Set).List() {
			slaPolicies = append(slaPolicies, policy.(string))
		}
		activityId, err = c.Backup().SLAPolicy().AssignVirtualDisk(ctx, &client.BackupAssignVirtualDiskRequest{
			VirtualDiskId: d.Id(),
			SLAPolicies:   slaPolicies,
		})
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual disk, %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual disk, %s", err)
		}
	}

	return computeVirtualDiskRead(ctx, d, meta)
}

func computeVirtualDiskRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	reader := readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
		disk, err := c.Compute().VirtualDisk().Read(ctx, d.Id())
		if err != nil {
			return nil, err
		}
		if disk == nil {
			return nil, nil
		}

		// Normalize the backup_sla_policies
		slaPolicies, err := c.Backup().SLAPolicy().List(ctx, &client.BackupSLAPolicyFilter{
			VirtualDiskId: d.Id(),
		})
		if err != nil {
			return nil, err
		}

		slaPoliciesIds := []string{}
		for _, slaPolicy := range slaPolicies {
			slaPoliciesIds = append(slaPoliciesIds, slaPolicy.ID)
		}

		sw.set("backup_sla_policies", slaPoliciesIds)

		return disk, nil
	})

	return reader(ctx, d, meta)
}

func computeVirtualDiskUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().VirtualDisk().Update(ctx, &client.UpdateVirtualDiskRequest{
		ID:          d.Id(),
		NewCapacity: d.Get("capacity").(int),
		DiskMode:    d.Get("disk_mode").(string),
	})
	if err != nil {
		return diag.Errorf("failed to update virtual disk: %s", err)
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("failed to update virtual disk, %s", err)
	}

	if d.HasChange("backup_sla_policies") {
		slaPolicies := []string{}
		for _, policy := range d.Get("backup_sla_policies").(*schema.Set).List() {
			slaPolicies = append(slaPolicies, policy.(string))
		}
		activityId, err = c.Backup().SLAPolicy().AssignVirtualDisk(ctx, &client.BackupAssignVirtualDiskRequest{
			VirtualDiskId: d.Id(),
			SLAPolicies:   slaPolicies,
		})
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual disk, %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual disk, %s", err)
		}
	}

	return computeVirtualDiskRead(ctx, d, meta)
}

func computeVirtualDiskDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().VirtualDisk().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete virtual disk: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete virtual disk, %s", err)
	}
	return nil
}
