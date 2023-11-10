package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceVirtualController() *schema.Resource {
	return &schema.Resource{
		Description: "Create and manage virtual controllers of a virtual machine.",

		CreateContext: computeVirtualControllerCreate,
		ReadContext:   computeVirtualControllerRead,
		UpdateContext: computeVirtualControllerUpdate,
		DeleteContext: computeVirtualControllerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			//In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"USB2", "USB3", "SCSI", "CD/DVD"}, false),
				Description:  "Can be one of : USB2, USB3, SCSI, CD/DVD",
			},
			"sub_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"BusLogic", "LSILogic", "LSILogicSAS", "ParaVirtual"}, false),
				Description:  "Can be one of : BusLogic, LSILogic, LSILogicSAS, ParaVirtual",
			},
			"iso_path": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"content_library_item_id"},
				Description:   "If exists, the datastore ISO path. (Conflicts with `content_library_item_id`)",
			},
			"content_library_item_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"iso_path"},
				Description:   "Content library item identifier. (Conflicts with `iso_path`)",
			},
			"connected": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Only compatible with CDROM controllers",
			},
			"mounted": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Only compatible with CDROM controllers",
			},

			//Out
			"hot_add_remove": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"shared_bus": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"label": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"summary": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"virtual_disks": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func computeVirtualControllerCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().VirtualController().Create(ctx, &client.CreateVirtualControllerRequest{
		VirtualMachineId: d.Get("virtual_machine_id").(string),
		Type:             d.Get("type").(string),
		SubType:          d.Get("sub_type").(string),
	})
	if err != nil {
		return diag.Errorf("the virtual controller could not be created: %s", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	setIdFromActivityConcernedItems(d, activity)
	if err != nil {
		return diag.Errorf("failed to create virtual controller, %s", err)
	}

	return computeVirtualControllerUpdate(ctx, d, meta)
}

func computeVirtualControllerRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	reader := readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
		controller, err := client.Compute().VirtualController().Read(ctx, d.Id())
		if err != nil {
			return nil, err
		}
		if controller == nil {
			return nil, nil
		}
		return controller, nil
	})

	return reader(ctx, d, meta)
}

func computeVirtualControllerUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	if d.HasChange("mounted") {
		var activityId string
		var err error
		if d.Get("mounted").(bool) {
			activityId, err = c.Compute().VirtualController().Mount(ctx, &client.MountVirtualControllerRequest{
				ID:                   d.Id(),
				IsoPath:              d.Get("iso_path").(string),
				ContentLibraryItemId: d.Get("content_library_item_id").(string),
			})
		} else {
			activityId, err = c.Compute().VirtualController().Unmount(ctx, d.Id())
		}
		if err != nil {
			return diag.Errorf("the virtual controller could not be connected: %s", err)
		}
		activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		setIdFromActivityConcernedItems(d, activity)
		if err != nil {
			return diag.Errorf("failed to connect virtual controller, %s", err)
		}
	}

	if d.HasChange("connected") {
		var activityId string
		var err error
		if d.Get("connected").(bool) {
			activityId, err = c.Compute().VirtualController().Connect(ctx, d.Id())
		} else {
			activityId, err = c.Compute().VirtualController().Disconnect(ctx, d.Id())
		}
		if err != nil {
			return diag.Errorf("the virtual controller could not be connected: %s", err)
		}
		activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		setIdFromActivityConcernedItems(d, activity)
		if err != nil {
			return diag.Errorf("failed to connect virtual controller, %s", err)
		}
	}

	return computeVirtualControllerRead(ctx, d, meta)
}

func computeVirtualControllerDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().VirtualController().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete virtual controller: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete virtual controller, %s", err)
	}
	return nil
}
