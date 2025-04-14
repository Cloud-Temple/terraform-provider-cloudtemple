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

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: computeNetworkRead,

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
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},
			"type": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.StringInSlice([]string{"Network", "DistributedVirtualPortgroup"}, false),
				ConflictsWith: []string{"id"},
			},
			"virtual_switch_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},
			"folder_id": {
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
			"virtual_machines_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"host_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"host_names": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

// computeNetworkRead lit un réseau et le mappe dans le state Terraform
func computeNetworkRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var network *client.Network
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		networks, err := c.Compute().Network().List(ctx, &client.NetworkFilter{
			Name:             name,
			MachineManagerId: d.Get("machine_manager_id").(string),
			DatacenterId:     d.Get("datacenter_id").(string),
			VirtualMachineId: d.Get("virtual_machine_id").(string),
			Type:             d.Get("type").(string),
			VirtualSwitchId:  d.Get("virtual_switch_id").(string),
			HostId:           d.Get("host_id").(string),
			FolderId:         d.Get("folder_id").(string),
			HostClusterId:    d.Get("host_cluster_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual network named %q: %s", name, err))
		}
		for _, n := range networks {
			if n.Name == name {
				network = n
				break
			}
		}
		if network == nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual network named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		network, err = c.Compute().Network().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if network == nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual network with id %q", id))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(network.ID)

	// Mapper les données en utilisant la fonction helper
	networkData := helpers.FlattenNetwork(network)

	// Définir les données dans le state
	for k, v := range networkData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
