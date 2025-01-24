package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceOpenIaasVirtualDisk() *schema.Resource {
	return &schema.Resource{
		CreateContext: openIaasVirtualDiskCreate,
		ReadContext:   openIaasVirtualDiskRead,
		//UpdateContext: openIaasVirtualDiskUpdate,
		DeleteContext: openIaasVirtualDiskDelete,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the virtual disk.",
				Required:    true,
				ForceNew:    true,
			},
			"size": {
				Type:        schema.TypeInt,
				Description: "The size of the virtual disk in bytes.",
				Required:    true,
				ForceNew:    true,
			},
			"mode": {
				Type:         schema.TypeString,
				Description:  "The mode of the virtual disk. Available values are RW (Read/Write) and RO (Read-Only).",
				ValidateFunc: validation.StringInSlice([]string{"RW", "RO"}, false),
				Required:     true,
				ForceNew:     true,
			},
			"storage_repository_id": {
				Type:         schema.TypeString,
				Description:  "The ID of the storage repository where the virtual disk is stored.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
				ForceNew:     true,
			},
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Description:  "The ID of the virtual machine to which the virtual disk is attached.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
				ForceNew:     true,
			},
			"bootable": {
				Type:        schema.TypeBool,
				Description: "Whether the virtual disk is bootable.",
				Optional:    true,
				Default:     false,
				ForceNew:    true,
			},

			// Out
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the virtual disk.",
			},
			"usage": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The usage of the virtual disk.",
			},
		},
	}
}

func openIaasVirtualDiskCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().OpenIaaS().VirtualDisk().Create(ctx, &client.OpenIaaSVirtualDiskCreateRequest{
		Name:                d.Get("name").(string),
		Size:                d.Get("size").(int),
		Mode:                d.Get("mode").(string),
		StorageRepositoryID: d.Get("storage_repository_id").(string),
		VirtualMachineID:    d.Get("virtual_machine_id").(string),
		Bootable:            d.Get("bootable").(bool),
	})
	if err != nil {
		return diag.Errorf("the virtual disk could not be created: %s", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions((ctx)))
	setIdFromActivityState(d, activity)
	if err != nil {
		return diag.Errorf("the virtual disk could not be created: %s", err)
	}

	return openIaasVirtualDiskRead(ctx, d, meta) // DevNote : Call update instead when it will be implemented
}

func openIaasVirtualDiskRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)
	virtualDisk, err := c.Compute().OpenIaaS().VirtualDisk().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("the virtual disk could not be read: %s", err)
	}
	if virtualDisk == nil {
		return diag.Errorf("the virtual disk could not be found: %s", err)
	}

	// Set the retrieved data to the schema
	sw := newStateWriter(d)
	sw.set("name", virtualDisk.Name)
	sw.set("size", virtualDisk.Size)
	sw.set("usage", virtualDisk.Usage)
	sw.set("storage_repository_id", virtualDisk.StorageRepository.ID)

	return sw.diags
}

func openIaasVirtualDiskUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO : Implement Detach on API to be able to change a virtual disk VM attachment

	// TODO : Implement this on Compute API
	// // Update the virtual disk properties if they have changed
	// if d.HasChange("name") || d.HasChange("size") || d.HasChange("mode") || d.HasChange("storage_repository_id") || d.HasChange("bootable") {
	// 	_, err := c.Compute().OpenIaaS().VirtualDisk().Update(ctx, d.Id(), &client.OpenIaaSVirtualDiskUpdateRequest{
	// 		Name:                d.Get("name").(string),
	// 		Size:                d.Get("size").(int),
	// 		Mode:                d.Get("mode").(string),
	// 		StorageRepositoryID: d.Get("storage_repository_id").(string),
	// 		Bootable:            d.Get("bootable").(bool),
	// 	})
	// 	if err != nil {
	// 		return diag.Errorf("failed to update virtual disk: %s", err)
	// 	}
	// }

	return openIaasVirtualDiskRead(ctx, d, meta)
}

func openIaasVirtualDiskDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().OpenIaaS().VirtualDisk().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete virtual disk: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete virtual disk, %s", err)
	}
	return nil
}
