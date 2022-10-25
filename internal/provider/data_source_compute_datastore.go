package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDatastore() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceDatastoreRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
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
	}
}

func dataSourceDatastoreRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	id := d.Get("id").(string)

	datastore, err := client.Compute().Datastore().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, id)

	hostNames := make([]interface{}, len(datastore.HostsNames))
	for i, hn := range datastore.HostsNames {
		hostNames[i] = hn
	}

	sw.set("name", datastore.Name)
	sw.set("moref", datastore.Moref)
	sw.set("max_capacity", datastore.MaxCapacity)
	sw.set("free_capacity", datastore.FreeCapacity)
	sw.set("accessible", datastore.Accessible)
	sw.set("maintenance_status", datastore.MaintenanceStatus)
	sw.set("unique_id", datastore.UniqueId)
	sw.set("machine_manager_id", datastore.MachineManagerId)
	sw.set("type", datastore.Type)
	sw.set("virtual_machines_number", datastore.VirtualMachinesNumber)
	sw.set("hosts_number", datastore.HostsNumber)
	sw.set("hosts_names", hostNames)
	sw.set("associated_folder", datastore.AssociatedFolder)

	return sw.diags
}
