package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDatastores() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceDatastoresRead,

		Schema: map[string]*schema.Schema{
			"datastores": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"moref": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"max_capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"free_capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"accessible": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"maintenance_status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"unique_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_machines_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"hosts_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"hosts_names": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"associated_folder": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceDatastoresRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	datastores, err := client.Compute().Datastore().List(ctx, "", "", "", "", "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(datastores))
	for i, d := range datastores {
		res[i] = map[string]interface{}{
			"id":                      d.ID,
			"name":                    d.Name,
			"moref":                   d.Moref,
			"max_capacity":            d.MaxCapacity,
			"free_capacity":           d.FreeCapacity,
			"accessible":              d.Accessible,
			"maintenance_status":      d.MaintenanceStatus,
			"unique_id":               d.UniqueId,
			"machine_manager_id":      d.MachineManagerId,
			"type":                    d.Type,
			"virtual_machines_number": d.VirtualMachinesNumber,
			"hosts_number":            d.HostsNumber,
			"hosts_names":             d.HostsNames,
			"associated_folder":       d.AssociatedFolder,
		}
	}

	sw := newStateWriter(d, "datastores")
	sw.set("datastores", res)

	return sw.diags
}
