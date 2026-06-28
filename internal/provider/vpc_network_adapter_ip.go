package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// This file holds the VPC static-IP (ip_address) helpers shared by the two
// network-adapter resources: cloudtemple_compute_iaas_opensource_network_adapter
// (#374) and cloudtemple_compute_network_adapter (VMware, #375). The decision
// logic is identical across both surfaces; the only genuine difference is how the
// target network's VPC-ness is read (OpenIaaS vs vCenter), which is injected as a
// networkVPCStatusFunc. (#379 unified the formerly duplicated vmware* copies.)

// networkVPCStatusFunc reports whether a network is VPC-backed. found is false
// when the network does not exist (the platform read returned no such network),
// so the caller can tell "not found" apart from "found but not a VPC network".
type networkVPCStatusFunc func(ctx context.Context, networkID string) (vpcBacked bool, found bool, err error)

// vpcStaticIPToPush decides the VPC static IP to send on an adapter update; it
// returns the address to push, or "" to leave the static IP untouched. It pushes
// ONLY when ip_address is explicitly configured, non-empty, the adapter is on a
// VPC-backed network, and the configured address diverges from the live one:
// re-sending an unchanged address would relocate the static IP to itself on every
// apply (a perpetual no-op activity), and ip_address has no meaning on a non-VPC
// network. The live static IP is resolved by MAC by the caller, because the
// platform does not echo it on the adapter object (#1854).
func vpcStaticIPToPush(ipConfigured bool, configuredIP, liveIP string, onVPC bool) string {
	if !ipConfigured || configuredIP == "" || !onVPC {
		return ""
	}
	if configuredIP == liveIP {
		return ""
	}
	return configuredIP
}

// adapterVPCStaticIP maps a by-MAC static IP read to the ip_address state value.
// A non-VPC adapter has no static IP; a VPC adapter with none registered yet
// (sip == nil — including the 403/absent the client maps to nil) also yields "".
func adapterVPCStaticIP(onVPC bool, sip *client.StaticIP) string {
	if !onVPC || sip == nil {
		return ""
	}
	return sip.IPAddress
}

// ensureVPCForIPAddress fails closed BEFORE any side effect when ip_address is
// explicitly configured but the target network_id is not a VPC-backed private
// network. ip_address only has meaning on a VPC network (it assigns the adapter's
// static IP there); on a plain network the platform ignores it, so the value
// could never be applied nor recorded and the plan would never converge.
// Validating the target network up front makes apply reject the bad config
// instead of creating/moving the adapter and only then failing. The network's
// VPC-ness is read through status, which differs per surface (OpenIaaS vs
// vCenter).
func ensureVPCForIPAddress(ctx context.Context, d *schema.ResourceData, status networkVPCStatusFunc) diag.Diagnostics {
	raw := d.GetRawConfig()
	if raw.IsNull() {
		return nil
	}
	if v := raw.GetAttr("ip_address"); v.IsNull() {
		return nil
	}
	ip := d.Get("ip_address").(string)
	if ip == "" {
		return nil
	}
	networkID := d.Get("network_id").(string)
	vpcBacked, found, err := status(ctx, networkID)
	if err != nil {
		return diag.Errorf("failed to read network %s to validate ip_address: %s", networkID, err)
	}
	return validateIPAddressTargetsVPC(ip, networkID, vpcBacked, found)
}

// validateIPAddressTargetsVPC is the fail-closed verdict of ensureVPCForIPAddress
// once the target network has been read: ip_address is only valid on an existing,
// VPC-backed network — a network that does not exist (found == false) or is not
// VPC-backed is rejected. Split out as a pure function so the fail-closed contract
// is unit-tested independently of the schema/raw-config plumbing.
func validateIPAddressTargetsVPC(ip, networkID string, vpcBacked, found bool) diag.Diagnostics {
	if !found {
		return diag.Errorf("network %s not found while validating ip_address", networkID)
	}
	if !vpcBacked {
		return diag.Errorf("ip_address %q is set but network %s is not a VPC-backed private network: ip_address requires network_id to reference a VPC private network — remove ip_address or use a VPC network", ip, networkID)
	}
	return nil
}
