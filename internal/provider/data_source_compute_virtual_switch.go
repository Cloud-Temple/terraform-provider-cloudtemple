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

func dataSourceVirtualSwitch() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific virtual switch from a vCenter infrastructure.",

		ReadContext: dataSourceVirtualSwitchRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},
			"host_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},

			// Out
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"folder_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// dataSourceVirtualSwitchRead lit un commutateur virtuel et le mappe dans le state Terraform
func dataSourceVirtualSwitchRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var virtualSwitch *client.VirtualSwitch
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		virtualSwitches, err := c.Compute().VirtualSwitch().List(ctx, &client.VirtualSwitchFilter{
			Name:             name,
			MachineManagerId: d.Get("machine_manager_id").(string),
			DatacenterId:     d.Get("datacenter_id").(string),
			HostClusterId:    d.Get("host_cluster_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual switch named %q: %s", name, err))
		}
		for _, dvs := range virtualSwitches {
			if dvs.Name == name {
				virtualSwitch = dvs
				break
			}
		}
		if virtualSwitch == nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual switch named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			virtualSwitch, err = c.Compute().VirtualSwitch().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if virtualSwitch == nil {
				return diag.FromErr(fmt.Errorf("failed to find virtual switch with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(virtualSwitch.ID)

	// Mapper les données en utilisant la fonction helper
	vsData := helpers.FlattenVirtualSwitch(virtualSwitch)

	// Définir les données dans le state
	for k, v := range vsData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
