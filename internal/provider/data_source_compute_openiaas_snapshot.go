package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific snapshot from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSSnapshotRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},

			// Out
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"create_time": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

// computeOpenIaaSSnapshotRead lit un snapshot OpenIaaS et le mappe dans le state Terraform
func computeOpenIaaSSnapshotRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var snapshot *client.OpenIaaSSnapshot
	var err error

	// Recherche par ID
	id := d.Get("id").(string)
	if id != "" {
		snapshot, err = c.Compute().OpenIaaS().Snapshot().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if snapshot == nil {
			return diag.FromErr(fmt.Errorf("failed to find snapshot with id %q", id))
		}
	} else {
		// Recherche par nom
		name := d.Get("name").(string)
		if name != "" {
			virtualMachineId := d.Get("virtual_machine_id").(string)
			if virtualMachineId == "" {
				return diag.FromErr(fmt.Errorf("virtual_machine_id is required when searching by name"))
			}

			snapshots, err := c.Compute().OpenIaaS().Snapshot().List(ctx, &client.OpenIaaSSnapshotFilter{
				VirtualMachineID: virtualMachineId,
			})
			if err != nil {
				return diag.FromErr(fmt.Errorf("failed to list snapshots: %s", err))
			}
			for _, s := range snapshots {
				if s.Name == name {
					snapshot = s
					break
				}
			}
			if snapshot == nil {
				return diag.FromErr(fmt.Errorf("failed to find snapshot named %q", name))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(snapshot.ID)

	// Mapper les données en utilisant la fonction helper
	snapshotData := helpers.FlattenOpenIaaSSnapshot(snapshot)

	// Définir les données dans le state
	for k, v := range snapshotData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
