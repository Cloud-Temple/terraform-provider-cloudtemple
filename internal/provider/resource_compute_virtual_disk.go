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

		CreateContext: computeVirtualDiskCreate,
		ReadContext:   computeVirtualDiskRead,
		UpdateContext: computeVirtualDiskUpdate,
		DeleteContext: computeVirtualDiskDelete,

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
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"provisioning_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"disk_mode": {
				Type:     schema.TypeString,
				Required: true,
			},
			"capacity": {
				Type:     schema.TypeInt,
				Required: true,
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
	setIdFromActivityConcernedItems(d, activity)
	if err != nil {
		return diag.Errorf("failed to create virtual disk, %s", err)
	}

	return computeVirtualDiskRead(ctx, d, meta)
}

func computeVirtualDiskRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	reader := readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
		return client.Compute().VirtualDisk().Read(ctx, d.Id())
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
