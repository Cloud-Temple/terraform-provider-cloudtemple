package provider

import (
	"context"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sethvargo/go-retry"
)

func resourceNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		Description: "Create and manage network adapter of a virtual machine.",

		CreateWithoutTimeout: computeNetworkAdapterCreate,
		ReadContext:          computeNetworkAdapterRead,
		UpdateContext:        computeNetworkAdapterUpdate,
		DeleteContext:        computeNetworkAdapterDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the virtual machine to which the network adapter will be attached.",
			},
			"network_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the network to which the adapter will be connected.",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The type of the network adapter. Supported types are defined by the Guest OS, usually E1000 or VMXNET3.",
			},
			"mac_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The MAC address of the network adapter. If not provided, a MAC address will be generated automatically.",
			},
			"auto_connect": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the network adapter should connect to the network automatically when the virtual machine is powered on.",
			},
			"connected": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the network adapter should be connected to the network. Defaults to true. ",
			},
			"ip_address": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IsIPv4Address,
				Description:  "The VPC static IP to assign to this adapter. Requires `network_id` to reference a VPC-backed network: when set, the adapter is given this address on the VPC; if omitted, the platform auto-assigns one (reflected here after apply). Mutable: changing it relocates the static IP. Setting it while `network_id` is not VPC-backed is rejected.",
			},

			// Out
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the network adapter.",
			},
			"mac_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of MAC address assignment. Possible values are MANUAL and GENERATED.",
			},
		},
	}
}

func computeNetworkAdapterCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Reject ip_address on a non-VPC network BEFORE creating anything.
	if diags := ensureVPCForIPAddress(ctx, d, vmwareNetworkVPCBacked(c)); diags != nil {
		return diags
	}

	activityId, err := c.Compute().NetworkAdapter().Create(ctx, &client.CreateNetworkAdapterRequest{
		VirtualMachineId: d.Get("virtual_machine_id").(string),
		MacAddress:       d.Get("mac_address").(string),
		NetworkId:        d.Get("network_id").(string),
		Type:             d.Get("type").(string),
		IPAddress:        d.Get("ip_address").(string),
	})
	if err != nil {
		return diag.Errorf("the network adapter could not be created: %s", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	setIdFromActivityConcernedItems(d, activity, "network_adapter")
	if err != nil {
		return diag.Errorf("failed to create network adapter, %s", err)
	}

	// We have to use update to set mac_type
	return computeNetworkAdapterUpdate(ctx, d, meta)
}

func computeNetworkAdapterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	// Récupérer l'adaptateur réseau par son ID
	networkAdapter, err := c.Compute().NetworkAdapter().Read(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if networkAdapter == nil {
		// A nil read is NOT a deletion: the client maps HTTP 403 to nil. We
		// never auto-remove the resource; we confirm liveness against a strict
		// VM-scoped listing and otherwise fail closed (#281).
		vmID := d.Get("virtual_machine_id").(string)
		return confirmVMwareDeviceOrKeep(ctx, d.Id(), "network adapter", "virtual machine", vmID,
			func(ctx context.Context) ([]string, error) {
				adapters, err := c.Compute().NetworkAdapter().ListStrict(ctx, &client.NetworkAdapterFilter{VirtualMachineID: vmID})
				if err != nil {
					return nil, err
				}
				ids := make([]string, 0, len(adapters))
				for _, a := range adapters {
					if a != nil {
						ids = append(ids, a.ID)
					}
				}
				return ids, nil
			})
	}

	// Mapper les données en utilisant la fonction helper
	adapterData := helpers.FlattenNetworkAdapter(networkAdapter)

	// Définir les données dans le state
	for k, v := range adapterData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	// Converge ip_address: the assigned VPC static IP is addressable only by MAC
	// (GET /vpc/v1/static_ips/mac/{mac}), not echoed on the adapter object
	// (#375 / #1854). Only a VPC-backed adapter has one; fail closed on a read
	// error rather than blanking ip_address (which would show a spurious diff).
	ipAddress := ""
	onVPC := networkAdapter.VPC != nil
	if onVPC && networkAdapter.MacAddress != "" {
		staticIP, err := c.VPC().StaticIP().ReadByMAC(ctx, networkAdapter.MacAddress)
		if err != nil {
			return diag.Errorf("failed to read the VPC static IP of network adapter %s: %s", d.Id(), err)
		}
		ipAddress = adapterVPCStaticIP(onVPC, staticIP)
	}
	if err := d.Set("ip_address", ipAddress); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func computeNetworkAdapterUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Reject ip_address on a non-VPC network BEFORE mutating the adapter.
	if diags := ensureVPCForIPAddress(ctx, d, vmwareNetworkVPCBacked(c)); diags != nil {
		return diags
	}

	activityId, err := c.Compute().NetworkAdapter().Update(ctx, &client.UpdateNetworkAdapterRequest{
		ID:           d.Id(),
		NewNetworkId: d.Get("network_id").(string),
		AutoConnect:  d.Get("auto_connect").(bool),
		MacAddress:   d.Get("mac_address").(string),
	})
	if err != nil {
		return diag.Errorf("failed to update network adapter, %s", err)
	}

	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
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
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			return err
		})
		if err != nil {
			return diag.Errorf("failed to %s network adapter, %s", msg, err)
		}
	}

	// VPC static IP reconciliation, AFTER the network/mac patch and against a
	// FRESH read: a same-apply move onto a VPC-backed network then shows the
	// adapter on the VPC, and the assigned IP is resolved by MAC (the platform
	// does not echo it on the adapter, #1854). Push only on a genuine divergence
	// so a no-op apply never relocates the static IP to itself.
	// mac_address is in the trigger because the VPC static IP is keyed BY MAC:
	// a MAC change re-targets the registration, so a configured ip_address must
	// be re-applied to the new MAC in the same apply (not just on ip/network change).
	if d.HasChange("ip_address") || d.HasChange("network_id") || d.HasChange("mac_address") {
		ipConfigured := false
		if raw := d.GetRawConfig(); !raw.IsNull() {
			if v := raw.GetAttr("ip_address"); !v.IsNull() {
				ipConfigured = true
			}
		}
		configuredIP := d.Get("ip_address").(string)
		if ipConfigured && configuredIP != "" {
			fresh, err := c.Compute().NetworkAdapter().Read(ctx, d.Id())
			if err != nil {
				return diag.Errorf("failed to read network adapter %s before VPC IP reconciliation: %s", d.Id(), err)
			}
			if fresh == nil {
				return diag.Errorf("network adapter %s not found", d.Id())
			}
			// A non-VPC target was already rejected up front; skip rather than
			// error here (rare mid-apply drift) since the patch already ran.
			if fresh.VPC != nil {
				staticIP, err := c.VPC().StaticIP().ReadByMAC(ctx, fresh.MacAddress)
				if err != nil {
					return diag.Errorf("failed to read the current VPC static IP of network adapter %s: %s", d.Id(), err)
				}
				liveIP := adapterVPCStaticIP(true, staticIP)
				if ip := vpcStaticIPToPush(true, configuredIP, liveIP, true); ip != "" {
					relocateActivity, err := c.Compute().NetworkAdapter().Update(ctx, &client.UpdateNetworkAdapterRequest{
						ID:           d.Id(),
						NewNetworkId: fresh.Network.ID,
						AutoConnect:  d.Get("auto_connect").(bool),
						MacAddress:   fresh.MacAddress,
						IPAddress:    ip,
					})
					if err != nil {
						return diag.Errorf("the VPC static IP of network adapter %s could not be set: %s", d.Id(), err)
					}
					if _, err := c.Activity().WaitForCompletion(ctx, relocateActivity, getWaiterOptions(ctx)); err != nil {
						return diag.Errorf("the VPC static IP of network adapter %s could not be set: %s", d.Id(), err)
					}
				}
			}
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
	if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete network adapter, %s", err)
	}
	return nil
}

// vmwareNetworkVPCBacked reads the vCenter network's VPC-ness for
// ensureVPCForIPAddress (the decision logic is shared with the OpenIaaS adapter;
// see vpc_network_adapter_ip.go). It is the only VMware-specific part of the
// ip_address pre-validation: the network read.
func vmwareNetworkVPCBacked(c *client.Client) networkVPCStatusFunc {
	return func(ctx context.Context, networkID string) (vpcBacked bool, found bool, err error) {
		network, err := c.Compute().Network().Read(ctx, networkID)
		if err != nil {
			return false, false, err
		}
		if network == nil {
			return false, false, nil
		}
		return network.VPC != nil, true, nil
	}
}
