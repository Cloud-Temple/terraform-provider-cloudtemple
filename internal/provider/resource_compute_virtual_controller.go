package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
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
				Description:  "The virtual machine ID the virtual controller is attached to.",
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"USB2", "USB3", "SCSI", "CD/DVD", "NVME"}, false),
				Description:  "Can be one of : USB2, USB3, SCSI, CD/DVD, NVME",
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
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the virtual controller supports hot add/remove.",
			},
			"shared_bus": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The shared bus type of the virtual controller.",
			},
			"label": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The label of the virtual controller.",
			},
			"summary": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The summary of the virtual controller.",
			},
			"virtual_disks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The virtual disks attached to the virtual controller.",

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
	setIdFromActivityConcernedItems(d, activity, "virtual_controller")
	if err != nil {
		return diag.Errorf("failed to create virtual controller, %s", err)
	}

	return computeVirtualControllerUpdate(ctx, d, meta)
}

func computeVirtualControllerRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	// Récupérer l'adaptateur réseau par son ID
	virtualController, err := c.Compute().VirtualController().Read(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if virtualController == nil {
		d.SetId("") // L'adaptateur n'existe plus, marquer la ressource comme supprimée
		return nil
	}

	// Mapper les données en utilisant la fonction helper
	controllerData := helpers.FlattenVirtualController(virtualController)

	// Définir les données dans le state
	for k, v := range controllerData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
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
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
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
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
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
