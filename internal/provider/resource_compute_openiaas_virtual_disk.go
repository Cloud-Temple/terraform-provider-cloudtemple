package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
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
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

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
			"internal_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internal ID of the virtual disk.",
			},
			"is_snapshot": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the virtual disk is a snapshot.",
			},
			"virtual_machines": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The virtual machines to which the virtual disk is attached.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the virtual machine.",
						},
						"read_only": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the virtual disk is attached in read-only mode.",
						},
					},
				},
			},
			"templates": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The templates to which the virtual disk is attached.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the template.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the template.",
						},
						"read_only": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the virtual disk is attached in read-only mode.",
						},
					},
				},
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
	var diags diag.Diagnostics

	// Récupérer le disque virtuel par son ID
	virtualDisk, err := c.Compute().OpenIaaS().VirtualDisk().Read(ctx, d.Id())
	if virtualDisk == nil || err != nil {
		// Si le disque virtuel n'existe pas, on définit l'ID à une chaîne vide
		// pour indiquer à Terraform que la ressource n'existe plus
		d.SetId("")
		return nil
	}

	// Mapper les données en utilisant la fonction helper
	diskData := helpers.FlattenOpenIaaSVirtualDisk(virtualDisk)

	// // Conserver l'ID de la VM existante si aucune VM n'est attachée
	// if len(virtualDisk.VirtualMachines) == 0 {
	// 	if vmID, ok := d.GetOk("virtual_machine_id"); ok {
	// 		diskData["virtual_machine_id"] = vmID.(string)
	// 	}
	// }

	// // Préserver la valeur bootable existante si elle est définie
	// if bootable, ok := d.GetOk("bootable"); ok {
	// 	diskData["bootable"] = bootable
	// }

	// Définir les données dans le state
	for k, v := range diskData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
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
