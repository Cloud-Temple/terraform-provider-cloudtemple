package provider

import (
	"context"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sethvargo/go-retry"
)

func resourceNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		Description: "",

		CreateContext: computeNetworkAdapterCreate,
		ReadContext:   computeNetworkAdapterRead,
		UpdateContext: computeNetworkAdapterUpdate,
		DeleteContext: computeNetworkAdapterDelete,

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"network_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"mac_address": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ComputedWhen: []string{"mac_type"},
			},
			"mac_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "MANUAL",
			},
			"auto_connect": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"connected": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			// Out
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func computeNetworkAdapterCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().NetworkAdapter().Create(ctx, &client.CreateNetworkAdapterRequest{
		VirtualMachineId: d.Get("virtual_machine_id").(string),
		MacAddress:       d.Get("mac_address").(string),
		NetworkId:        d.Get("network_id").(string),
		Type:             d.Get("type").(string),
	})
	if err != nil {
		return diag.Errorf("the network adapter could not be created: %s", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityId)
	if err != nil {
		return diag.Errorf("failed to create network adapter, %s", err)
	}

	d.SetId(activity.ConcernedItems[0].ID)

	// We have to use update to set mac_type
	return computeNetworkAdapterUpdate(ctx, d, meta)
}

func computeNetworkAdapterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	reader := readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
		return client.Compute().NetworkAdapter().Read(ctx, d.Id())
	})

	return reader(ctx, d, meta)
}

func computeNetworkAdapterUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	macType := d.Get("mac_type").(string)
	macAddress := d.Get("mac_address").(string)
	if macType == "ASSIGNED" {
		macAddress = ""
	}
	activityId, err := c.Compute().NetworkAdapter().Update(ctx, &client.UpdateNetworkAdapterRequest{
		ID:           d.Id(),
		NewNetworkId: d.Get("network_id").(string),
		AutoConnect:  d.Get("auto_connect").(bool),
		MacAddress:   macAddress,
		MacType:      macType,
	})
	_, err = c.Activity().WaitForCompletion(ctx, activityId)
	if err != nil {
		return diag.Errorf("failed to update network adapter, %s", err)
	}

	if d.HasChange("connected") {
		var msg string
		var action func(context.Context, string) (string, error)
		if d.Get("connected").(bool) {
			msg = "connect"
			action = c.Compute().NetworkAdapter().Connect
		} else {
			msg = "disconnect"
			action = c.Compute().NetworkAdapter().Disconnect
		}

		// Connecting a network adapter can fail right after the VM has been powered
		// on so we retry here until we reach the timeout
		b := retry.NewFibonacci(1 * time.Second)
		b = retry.WithCappedDuration(20*time.Second, b)

		err = retry.Do(ctx, b, func(ctx context.Context) error {
			activityId, err = action(ctx, d.Id())
			if err != nil {
				return err
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId)
			return err
		})
		if err != nil {
			return diag.Errorf("failed to %s network adapter, %s", msg, err)
		}
	}

	return computeNetworkAdapterRead(ctx, d, meta)
}

func computeNetworkAdapterDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().NetworkAdapter().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete network adapter: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId); err != nil {
		return diag.Errorf("failed to delete network adapter, %s", err)
	}
	return nil
}
