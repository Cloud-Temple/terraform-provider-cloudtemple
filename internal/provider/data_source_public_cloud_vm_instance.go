package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// publicCloudVMInstanceRefSchema returns the Computed {id, name} nested block used
// for the resolved availability zone, template, instance family and backup policy.
func publicCloudVMInstanceRefSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: description,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id":   {Type: schema.TypeString, Computed: true, Description: "The unique identifier."},
				"name": {Type: schema.TypeString, Computed: true, Description: "The name."},
			},
		},
	}
}

// publicCloudVMInstanceComputedAttributes returns the Computed attributes of a VM
// instance, shared by the single and list datasources (the list nests them, plus
// a Computed `id`, inside each element). `id` is not included here because the
// single datasource takes it as a required input.
func publicCloudVMInstanceComputedAttributes() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name":                  {Type: schema.TypeString, Computed: true, Description: "The name of the virtual machine."},
		"status":                {Type: schema.TypeString, Computed: true, Description: "The current status of the VM (e.g. `running`, `stopped`)."},
		"availability_zone":     publicCloudVMInstanceRefSchema("The resolved availability zone."),
		"template":              publicCloudVMInstanceRefSchema("The resolved OS template."),
		"instance_family":       publicCloudVMInstanceRefSchema("The resolved instance family."),
		"vcpu":                  {Type: schema.TypeInt, Computed: true, Description: "The number of vCPUs."},
		"ram_gb":                {Type: schema.TypeInt, Computed: true, Description: "The amount of RAM in GB."},
		"disks_size_gb":         {Type: schema.TypeInt, Computed: true, Description: "The total size of the VM's disks (system + data) in GB."},
		"backup_policy":         publicCloudVMInstanceRefSchema("The applied backup policy (empty when none)."),
		"guest_tools_installed": {Type: schema.TypeBool, Computed: true, Description: "Whether the guest tools are installed."},
		"created_at":            {Type: schema.TypeString, Computed: true, Description: "The creation date of the VM (RFC3339)."},
		"updated_at":            {Type: schema.TypeString, Computed: true, Description: "The last update date of the VM (RFC3339)."},
	}
}

func dataSourcePublicCloudVMInstance() *schema.Resource {
	s := publicCloudVMInstanceComputedAttributes()
	s["id"] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.IsUUID,
		Description:  "The ID of the VM instance to retrieve.",
	}
	return &schema.Resource{
		Description: "Used to retrieve a single Public Cloud VM instance by `id`.",
		ReadContext: publicCloudVMInstanceRead,
		Schema:      s,
	}
}

func publicCloudVMInstanceRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	id := d.Get("id").(string)

	vm, err := c.PublicCloudVM().Instance().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if vm == nil {
		return diag.FromErr(fmt.Errorf("failed to find VM instance with id %q", id))
	}

	d.SetId(vm.ID)
	for k, v := range helpers.FlattenPublicCloudVMInstance(vm) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}
