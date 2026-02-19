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
		UpdateContext: openIaasVirtualDiskUpdate,
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
			},
			"size": {
				Type:        schema.TypeInt,
				Description: "The size of the virtual disk in bytes.",
				Required:    true,
			},
			"mode": {
				Type:         schema.TypeString,
				Description:  "The mode of the virtual disk. Available values are RW (Read/Write) and RO (Read-Only).",
				ValidateFunc: validation.StringInSlice([]string{"RW", "RO"}, false),
				Required:     true,
			},
			"storage_repository_id": {
				Type:         schema.TypeString,
				Description:  "The ID of the storage repository where the virtual disk is stored.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Description:  "The ID of the virtual machine to which the virtual disk is attached.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"bootable": {
				Type:        schema.TypeBool,
				Description: "Whether the virtual disk is bootable.",
				Optional:    true,
				Default:     false,
			},
			"connected": {
				Type:        schema.TypeBool,
				Description: "Whether the virtual disk should be connected to the virtual machine. Only applicable when virtual_machine_id is set.",
				Optional:    true,
				Default:     true,
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
						"connected": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the virtual disk is currently connected to the virtual machine.",
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

	// If connected is false and virtual_machine_id is set, disconnect the disk after creation
	if !d.Get("connected").(bool) && d.Get("virtual_machine_id").(string) != "" {
		activityId, err := c.Compute().OpenIaaS().VirtualDisk().Disconnect(ctx, d.Id(), &client.OpenIaaSVirtualDiskConnectionRequest{
			VirtualMachineID: d.Get("virtual_machine_id").(string),
		})
		if err != nil {
			return diag.Errorf("failed to disconnect virtual disk after creation: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to disconnect virtual disk after creation: %s", err)
		}
	}

	return openIaasVirtualDiskRead(ctx, d, meta)
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
	vmID := d.Get("virtual_machine_id").(string)
	diskData := helpers.FlattenOpenIaaSVirtualDisk(virtualDisk, vmID)

	// Définir les données dans le state
	for k, v := range diskData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func openIaasVirtualDiskUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)

	disk, err := c.Compute().OpenIaaS().VirtualDisk().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read virtual disk state before update: %s", err)
	}

	// Handle connection state changes
	if d.HasChange("connected") && d.Get("virtual_machine_id").(string) != "" {
		connected := d.Get("connected").(bool)
		vmID := d.Get("virtual_machine_id").(string)

		if connected {
			// Connect the disk to the VM
			activityId, err := c.Compute().OpenIaaS().VirtualDisk().Connect(ctx, d.Id(), &client.OpenIaaSVirtualDiskConnectionRequest{
				VirtualMachineID: vmID,
			})
			if err != nil {
				return diag.Errorf("failed to connect virtual disk: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to connect virtual disk: %s", err)
			}
		} else {
			// Disconnect the disk from the VM
			activityId, err := c.Compute().OpenIaaS().VirtualDisk().Disconnect(ctx, d.Id(), &client.OpenIaaSVirtualDiskConnectionRequest{
				VirtualMachineID: vmID,
			})
			if err != nil {
				return diag.Errorf("failed to disconnect virtual disk: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to disconnect virtual disk: %s", err)
			}
		}
	}

	// Virtual disk attachment cycle management
	if d.HasChange("virtual_machine_id") || d.HasChange("bootable") || d.HasChange("mode") {
		old, new := d.GetChange("virtual_machine_id")

		// Check if the disk is already attached to the new VM to avoid errors
		alreadyAttached := false
		for _, vm := range disk.VirtualMachines {
			if vm.ID == new {
				alreadyAttached = true
			}
		}

		if old != "" && len(disk.VirtualMachines) > 0 {
			activityId, err := c.Compute().OpenIaaS().VirtualDisk().Detach(ctx, d.Id(), &client.OpenIaaSVirtualDiskDetachRequest{
				VirtualMachineID: old.(string),
			})
			if err != nil {
				return diag.Errorf("failed to detach virtual disk: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to detach virtual disk: %s", err)
			}
		}

		if new != "" && !alreadyAttached {
			activityId, err := c.Compute().OpenIaaS().VirtualDisk().Attach(ctx, d.Id(), &client.OpenIaaSVirtualDiskAttachRequest{
				VirtualMachineID: d.Get("virtual_machine_id").(string),
				Bootable:         d.Get("bootable").(bool),
				Mode:             d.Get("mode").(string),
			})
			if err != nil {
				return diag.Errorf("failed to attach virtual disk: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to attach virtual disk: %s", err)
			}
		}
	}

	if d.HasChange("name") || d.HasChange("size") {
		var connectedVMs []string
		for _, vm := range disk.VirtualMachines {
			if vm.Connected {
				connectedVMs = append(connectedVMs, vm.ID)

				// Disconnect disk from VM before resizing
				activityId, err := c.Compute().OpenIaaS().VirtualDisk().Disconnect(ctx, d.Id(), &client.OpenIaaSVirtualDiskConnectionRequest{
					VirtualMachineID: vm.ID,
				})
				if err != nil {
					return diag.Errorf("failed to disconnect virtual disk from VM %s before update: %s", vm.ID, err)
				}
				_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
				if err != nil {
					return diag.Errorf("failed to disconnect virtual disk from VM %s before update: %s", vm.ID, err)
				}
			}
		}

		// Update the disk (e.g., resize)
		activityId, err := c.Compute().OpenIaaS().VirtualDisk().Update(ctx, d.Id(), &client.OpenIaaSVirtualDiskUpdateRequest{
			Name: d.Get("name").(string),
			Size: d.Get("size").(int),
		})
		if err != nil {
			return diag.Errorf("failed to update virtual disk: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to update virtual disk: %s", err)
		}

		// Reconnect the disk to previously connected VMs
		for _, vmID := range connectedVMs {
			activityId, err := c.Compute().OpenIaaS().VirtualDisk().Connect(ctx, d.Id(), &client.OpenIaaSVirtualDiskConnectionRequest{
				VirtualMachineID: vmID,
			})
			if err != nil {
				return diag.Errorf("failed to reconnect virtual disk to VM %s after update: %s", vmID, err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to reconnect virtual disk to VM %s after update: %s", vmID, err)
			}
		}
	}

	if d.HasChange("storage_repository_id") {
		activityId, err := c.Compute().OpenIaaS().VirtualDisk().Relocate(ctx, d.Id(), &client.OpenIaaSVirtualDiskRelocateRequest{
			StorageRepositoryID: d.Get("storage_repository_id").(string),
		})
		if err != nil {
			return diag.Errorf("failed to relocate virtual disk: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to relocate virtual disk: %s", err)
		}
	}

	return openIaasVirtualDiskRead(ctx, d, meta)
}

func openIaasVirtualDiskDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)

	// Disconnect disk before delete
	disk, err := c.Compute().OpenIaaS().VirtualDisk().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read virtual disk state before delete: %s", err)
	}
	for _, vm := range disk.VirtualMachines {
		if vm.Connected {
			activityId, err := c.Compute().OpenIaaS().VirtualDisk().Disconnect(ctx, d.Id(), &client.OpenIaaSVirtualDiskConnectionRequest{
				VirtualMachineID: vm.ID,
			})
			if err != nil {
				return diag.Errorf("failed to disconnect virtual disk from VM %s before delete: %s", vm.ID, err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to disconnect virtual disk from VM %s before delete: %s", vm.ID, err)
			}
		}
	}

	activityId, err := c.Compute().OpenIaaS().VirtualDisk().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete virtual disk: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete virtual disk, %s", err)
	}
	return nil
}
