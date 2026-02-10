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

func dataSourceHost() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific host.",

		ReadContext: computeHostRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the host to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				Description:   "The name of the host to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the machine manager this host belongs to.",
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the datacenter this host belongs to.",
			},
			"host_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the host cluster this host belongs to.",
			},
			"datastore_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the datastore this host belongs to.",
			},

			// Out
			"moref": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The managed object reference ID of the host.",
			},
			"metrics": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Resource metrics for the host.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"esx": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "ESX hypervisor information.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"version": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ESX version.",
									},
									"build": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The ESX build number.",
									},
									"full_name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The full name of the ESX version.",
									},
								},
							},
						},
						"cpu": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "CPU metrics for the host.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"overall_cpu_usage": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The overall CPU usage in MHz.",
									},
									"cpu_mhz": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The CPU frequency in MHz.",
									},
									"cpu_cores": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of CPU cores.",
									},
									"cpu_threads": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of CPU threads.",
									},
								},
							},
						},
						"memory": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Memory metrics for the host.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"memory_size": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The total memory size in bytes.",
									},
									"memory_usage": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The memory usage in bytes.",
									},
								},
							},
						},
						"maintenance_status": {
							Type:        schema.TypeBool,
							Computed:    true,
							Deprecated:  "Use maintenance_mode instead. This field will be removed in a future version.",
							Description: "Whether the host is in maintenance mode. Deprecated: use maintenance_mode instead.",
						},
						"maintenance_mode": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the host is in maintenance mode.",
						},
						"uptime": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The host uptime in seconds.",
						},
						"connected": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the host is connected.",
						},
					},
				},
			},
			"virtual_machines": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of virtual machines running on this host.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the virtual machine.",
						},
					},
				},
			},
		},
	}
}

// computeHostRead lit un hôte et le mappe dans le state Terraform
func computeHostRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var host *client.Host
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		hosts, err := c.Compute().Host().List(ctx, &client.HostFilter{
			MachineManagerID: d.Get("machine_manager_id").(string),
			DatacenterID:     d.Get("datacenter_id").(string),
			HostClusterID:    d.Get("host_cluster_id").(string),
			DatastoreID:      d.Get("datastore_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find host named %q: %s", name, err))
		}
		for _, h := range hosts {
			if h.Name == name {
				host = h
				break
			}
		}
		if host == nil {
			return diag.FromErr(fmt.Errorf("failed to find host named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			host, err = c.Compute().Host().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if host == nil {
				return diag.FromErr(fmt.Errorf("failed to find host with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(host.ID)

	// Mapper les données en utilisant la fonction helper
	hostData := helpers.FlattenHost(host)

	// Définir les données dans le state
	for k, v := range hostData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
