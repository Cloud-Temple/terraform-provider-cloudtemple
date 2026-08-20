package client

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// This file is an OPT-IN, human-gated live probe for the "VPC network on the
// inline os_network_adapter block" question (#376, #473). With
// CT_VPC_VMCREATE_PROBE unset it Skips immediately, so a plain `go test ./...`
// touches nothing.
//
// It exists because a spec is never proof (.clinerules/WORKFLOW_ENGINEERING.md
// §5): issue #376 records the Compute VM-create endpoints as NOT accepting a
// per-adapter `ipAddress`, and issue #473 records the VMI inline block as
// refusing VPC networks by provider decision. Only the live API can say which of
// those is still true.
//
// SAFETY: the probe FAIL-CLOSES unless the configured host is exactly
// vpcCreateProbeHost (the DEV broker) — DefaultConfig() falls back to the
// PRODUCTION host, so a forgotten CLOUDTEMPLE_HTTP_ADDR must never be probed.
// Phase 1 is strictly read-only (GET only). Every write in this file lives behind
// a SECOND flag (CT_VPC_VMCREATE_PROBE_WRITE) and is covered by a t.Cleanup
// registered BEFORE the write, so a mid-probe Fatal cannot leave an orphan.
//
// Cleanup covers TWO planes, because a VM delete is not the only thing that can
// leak here:
//   - compute: a name-based VM sweep on the exact probe name;
//   - IPAM: a snapshot-difference sweep of the target private network
//     (snapshotStaticIPs before the write, sweepProbeStaticIPs after). This is
//     what catches the two cases a per-address sweep cannot: an AUTO-ASSIGNED
//     address the probe never learns, and a create activity that registered the
//     address and then failed before the VM became listable.
//
// The IPAM sweep deletes ONLY on strict positive ownership evidence (§5): the
// registration is attached to a VM carrying the probe's exact name, or its MAC
// address is one an adapter of the probe's own VM came up with. Neither an
// unattached registration nor a matching ADDRESS counts as evidence: the probe
// picks a currently-free address, but on a shared tenant another operator can
// register that very address between the baseline and the sweep, and deleting on
// equality alone would destroy their live registration. If the pre-write baseline
// could not be taken at all, the destructive path is disabled outright rather than
// run against an empty baseline.
//
// Residual limitation, stated rather than hidden: an address leaked by a create
// activity that failed before the probe could observe the adapter's MAC is
// therefore REPORTED, not reclaimed. Every such path logs "MANUAL CLEANUP NEEDED"
// loudly rather than passing silently. Trading a possible manual cleanup for the
// risk of deleting someone else's live address is the correct direction.
//
// Human GO: requested and granted by Paul Besret on 2026-08-20 for the DEV
// broker (api.shiva.dev.ctlabs.me), for the purpose of establishing what the
// VM-create endpoints accept on a VPC network.
const (
	vpcCreateProbeHost = "api.shiva.dev.ctlabs.me"
)

// requireVPCCreateProbeClient applies the two-part gate (opt-in flag + host
// allowlist) shared by every phase of this probe.
func requireVPCCreateProbeClient(t *testing.T) (*Client, *Config) {
	t.Helper()
	if os.Getenv("CT_VPC_VMCREATE_PROBE") != "1" {
		t.Skip("opt-in live probe; set CT_VPC_VMCREATE_PROBE=1 (read-only) and CT_VPC_VMCREATE_PROBE_WRITE=1 (controlled create/delete)")
	}
	cfg := DefaultConfig()
	if cfg.Address != vpcCreateProbeHost {
		t.Fatalf("refusing to run: CLOUDTEMPLE_HTTP_ADDR=%q but this probe is authorised ONLY for %q", cfg.Address, vpcCreateProbeHost)
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, cfg
}

// TestVPCVMCreateLiveProbePhase1 is READ-ONLY. It inventories the VPC-backed
// networks visible on each of the three VM surfaces and pins two facts a spec
// cannot give us:
//
//   - whether any VPC-backed network is reachable at all on this tenant (without
//     one, no VPC create can ever be validated);
//   - whether the `vpc` block is emitted by the SINGLE GET as well as by the
//     listing on the VMware surface. The existing VMware pre-validation
//     (vmwareNetworkVPCBacked) reads the single GET, while the client comment
//     only claims the listing was verified live — if the single GET omits `vpc`,
//     that pre-validation already rejects legitimate VPC networks.
func TestVPCVMCreateLiveProbePhase1(t *testing.T) {
	c, cfg := requireVPCCreateProbeClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	tok, err := c.Token(ctx)
	if err != nil {
		t.Fatalf("auth failed (check CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID): %v", err)
	}
	t.Logf("AUTH ok: host=%s tenant=%s user=%s", cfg.Address, tok.TenantID(), tok.UserID())

	rawGet := func(path string, args ...interface{}) (int, string) {
		r := c.newRequest("GET", path, args...)
		resp, err := c.doRequest(ctx, r)
		if err != nil {
			return 0, "REQUEST ERROR: " + err.Error()
		}
		defer closeResponseBody(resp)
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	// ---- VMI / public cloud -------------------------------------------------
	vmiNets, err := c.PublicCloudVM().Network().List(ctx)
	if err != nil {
		t.Fatalf("VMI Network().List failed: %v", err)
	}
	var vmiVPC, vmiPB []*PublicCloudVMNetwork
	for _, n := range vmiNets {
		if n == nil {
			continue
		}
		if n.VPC != nil {
			vmiVPC = append(vmiVPC, n)
		} else {
			vmiPB = append(vmiPB, n)
		}
	}
	t.Logf("VMI  /vm_instances/v1/networks: %d networks total — %d VPC-backed, %d Private Backbone", len(vmiNets), len(vmiVPC), len(vmiPB))
	for _, n := range vmiVPC {
		pn := "<nil>"
		if n.VPC.PrivateNetwork != nil {
			pn = n.VPC.PrivateNetwork.ID + " (" + n.VPC.PrivateNetwork.Name + ")"
		}
		t.Logf("VMI  VPC network: id=%s name=%q vpc={id:%s name:%q privateNetwork:%s}", n.ID, n.Name, n.VPC.ID, n.VPC.Name, pn)
	}
	if len(vmiVPC) > 0 {
		// Does the SINGLE GET expose `vpc` too? (the guard reads the single GET)
		st, body := rawGet("/vm_instances/v1/networks/%s", vmiVPC[0].ID)
		t.Logf("VMI  single GET networks/%s -> HTTP %d, vpc-key-present=%t, body=%s",
			vmiVPC[0].ID, st, strings.Contains(body, "\"vpc\""), truncateProbeBody(body))
	}

	// ---- OpenIaaS ----------------------------------------------------------
	oiNets, err := c.Compute().OpenIaaS().Network().List(ctx, nil)
	if err != nil {
		t.Logf("OpenIaaS Network().List failed: %v", err)
	} else {
		var oiVPC []*OpenIaaSNetwork
		for _, n := range oiNets {
			if n != nil && n.VPC != nil {
				oiVPC = append(oiVPC, n)
			}
		}
		t.Logf("OI   /compute/v1/open_iaas/networks: %d networks total — %d VPC-backed", len(oiNets), len(oiVPC))
		for _, n := range oiVPC {
			t.Logf("OI   VPC network: id=%s name=%q vpc={id:%s name:%q privateNetwork:{id:%s name:%q} staticIp=%q}",
				n.ID, n.Name, n.VPC.ID, n.VPC.Name, n.VPC.PrivateNetwork.ID, n.VPC.PrivateNetwork.Name, n.VPC.StaticIPAddress)
		}
		if len(oiVPC) > 0 {
			st, body := rawGet("/compute/v1/open_iaas/networks/%s", oiVPC[0].ID)
			t.Logf("OI   single GET networks/%s -> HTTP %d, vpc-key-present=%t, body=%s",
				oiVPC[0].ID, st, strings.Contains(body, "\"vpc\""), truncateProbeBody(body))
		}
	}

	// ---- VMware / vCenter --------------------------------------------------
	vwNets, err := c.Compute().Network().List(ctx, nil)
	if err != nil {
		t.Logf("VMware Network().List failed: %v", err)
	} else {
		var vwVPC []*Network
		for _, n := range vwNets {
			if n != nil && n.VPC != nil {
				vwVPC = append(vwVPC, n)
			}
		}
		t.Logf("VW   /compute/v1/vcenters/networks: %d networks total — %d VPC-backed", len(vwNets), len(vwVPC))
		for _, n := range vwVPC {
			t.Logf("VW   VPC network: id=%s name=%q moref=%s vpc={id:%s name:%q privateNetwork:{id:%s name:%q}}",
				n.ID, n.Name, n.Moref, n.VPC.ID, n.VPC.Name, n.VPC.PrivateNetwork.ID, n.VPC.PrivateNetwork.Name)
		}
		if len(vwVPC) > 0 {
			// F.6: the pre-validation reads the single GET; prove `vpc` survives there.
			st, body := rawGet("/compute/v1/vcenters/networks/%s", vwVPC[0].ID)
			t.Logf("VW   single GET networks/%s -> HTTP %d, vpc-key-present=%t, body=%s",
				vwVPC[0].ID, st, strings.Contains(body, "\"vpc\""), truncateProbeBody(body))
			single, rerr := c.Compute().Network().Read(ctx, vwVPC[0].ID)
			switch {
			case rerr != nil:
				t.Logf("VW   Read(%s) errored: %v", vwVPC[0].ID, rerr)
			case single == nil:
				t.Logf("VW   Read(%s) returned nil (absent)", vwVPC[0].ID)
			default:
				t.Logf("VW   Read(%s) decoded VPC != nil = %t  <-- vmwareNetworkVPCBacked depends on this", vwVPC[0].ID, single.VPC != nil)
			}
		}
	}

	// ---- VPC plane ---------------------------------------------------------
	vpcs, err := c.VPC().VPC().List(ctx)
	if err != nil {
		t.Logf("VPC().VPC().List failed: %v", err)
	} else {
		t.Logf("VPCP /vpc/v1/vpc: %d VPC(s)", len(vpcs))
		for _, v := range vpcs {
			if v != nil {
				t.Logf("VPCP vpc: id=%s name=%q", v.ID, v.Name)
			}
		}
	}
	pns, err := c.VPC().PrivateNetwork().List(ctx, nil)
	if err != nil {
		t.Logf("VPC().PrivateNetwork().List failed: %v", err)
	} else {
		t.Logf("VPCP /vpc/v1/private_networks: %d private network(s)", len(pns))
		for _, p := range pns {
			if p != nil {
				b, _ := json.Marshal(p)
				t.Logf("VPCP privateNetwork: %s", truncateProbeBody(string(b)))
			}
		}
	}

	if len(vmiVPC) == 0 {
		t.Log("VERDICT phase 1: NO VPC-backed network visible on the VMI surface — a VMI VPC create cannot be proven on this tenant")
	}
	t.Log("PHASE 1 DONE (read-only)")
}

// truncateProbeBody keeps probe logs readable without hiding the shape.
func truncateProbeBody(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "  ", " ")
	if len(s) > 900 {
		return s[:900] + "…[truncated]"
	}
	return s
}

// MEASURED, not assumed (DEV, 2026-08-20): POST /vm_instances/v1/virtual_machines
// answers 201 + Location even for a body whose availabilityZoneId / imageId /
// instanceFamilyId / backupPolicyId are all-zero UUIDs — the endpoint is
// ASYNCHRONOUS and does not validate identifiers before accepting. The refusal
// appears in the activity, not in the response. POST /compute/v1/open_iaas/virtual_machines
// behaves comparably (it answered 404 for an unknown template, again after
// accepting the body shape).
//
// Consequence, recorded here so nobody re-attempts it: a differential
// "unknown-key vs known-key" schema probe is INCONCLUSIVE on these endpoints,
// because neither validates the body synchronously and neither echoes an unknown
// key back. An earlier revision of this file shipped such a probe; it produced no
// evidence and performed accepted live POSTs with no activity wait and no cleanup
// net, so it was removed rather than hardened. Only a real create against a real
// image/template answers whether ipAddress is honoured — which is what the two
// write phases below do.

const vpcCreateProbeVMName = "tf-vpc-vmcreate-probe"

// vpcProbeTarget is the VPC substrate the write phase runs against.
type vpcProbeTarget struct {
	NetworkID        string // the COMPUTE-plane network id (what os_network_adapter takes)
	NetworkName      string
	PrivateNetworkID string // the VPC-plane private-network id (what /vpc/v1 takes)
	CIDR             string
	FreeIP           string // an address in CIDR not currently registered
	TakenCount       int
}

// pickVMIVPCTarget resolves a VPC-backed VMI network and a free static IP inside
// its private network. It deliberately prefers the private network with the
// FEWEST registered static IPs, to keep the blast radius of a live probe minimal.
func pickVMIVPCTarget(ctx context.Context, t *testing.T, c *Client) *vpcProbeTarget {
	t.Helper()
	nets, err := c.PublicCloudVM().Network().List(ctx)
	if err != nil {
		t.Fatalf("Network().List failed: %v", err)
	}
	pns, err := c.VPC().PrivateNetwork().List(ctx, nil)
	if err != nil {
		t.Fatalf("PrivateNetwork().List failed: %v", err)
	}
	cidrByPN := map[string]string{}
	for _, p := range pns {
		if p != nil {
			cidrByPN[p.ID] = p.IPAddress
		}
	}

	var best *vpcProbeTarget
	for _, n := range nets {
		if n == nil || n.VPC == nil || n.VPC.PrivateNetwork == nil {
			continue
		}
		pnID := n.VPC.PrivateNetwork.ID
		cidr := cidrByPN[pnID]
		// Only a /24 gives a predictable, small host range to pick from.
		if !strings.HasSuffix(cidr, "/24") {
			continue
		}
		ips, err := c.VPC().StaticIP().List(ctx, pnID, nil)
		if err != nil {
			t.Logf("  StaticIP().List(%s) failed, skipping: %v", pnID, err)
			continue
		}
		taken := map[string]bool{}
		for _, ip := range ips {
			if ip != nil {
				taken[ip.IPAddress] = true
			}
		}
		free := firstFreeHostInSlash24(cidr, taken)
		if free == "" {
			continue
		}
		cand := &vpcProbeTarget{
			NetworkID: n.ID, NetworkName: n.Name, PrivateNetworkID: pnID,
			CIDR: cidr, FreeIP: free, TakenCount: len(taken),
		}
		if best == nil || cand.TakenCount < best.TakenCount {
			best = cand
		}
	}
	if best == nil {
		t.Skip("no VPC-backed VMI network with a /24 private network and a free address; cannot run the write probe")
	}
	t.Logf("TARGET vpc network %s (%q) — privateNetwork=%s cidr=%s registeredStaticIPs=%d chosenFreeIP=%s",
		best.NetworkID, best.NetworkName, best.PrivateNetworkID, best.CIDR, best.TakenCount, best.FreeIP)
	return best
}

// firstFreeHostInSlash24 returns the highest-numbered unused host address in the
// upper part of a /24 (.240 down to .200). The upper range is used on purpose:
// platform auto-assignment hands out low addresses first, so picking high
// minimises the chance of racing a concurrent allocation.
func firstFreeHostInSlash24(cidr string, taken map[string]bool) string {
	base := strings.TrimSuffix(cidr, "/24")
	octets := strings.Split(base, ".")
	if len(octets) != 4 {
		return ""
	}
	prefix := octets[0] + "." + octets[1] + "." + octets[2] + "."
	for last := 240; last >= 200; last-- {
		cand := prefix + itoaProbe(last)
		if !taken[cand] {
			return cand
		}
	}
	return ""
}

func itoaProbe(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// TestVPCVMCreateLiveProbePhase2VMI is the DECISIVE experiment for the VMI
// surface. It answers, with live evidence, the questions no spec can:
//
//	Q1 does POST /vm_instances/v1/virtual_machines accept a VPC networkId in
//	   networkInterfaces[] at all?  (the whole basis of the #473 guard)
//	Q2 is networkInterfaces[].ipAddress HONOURED on that path — i.e. does the
//	   created adapter end up with the address we asked for?
//	Q3 without ipAddress, does the platform auto-assign a static IP on a VPC
//	   network?  (decides whether ip_address must be required)
//	Q4 does deleting the VM RECLAIM the static-IP registration, or does it leak?
//
// It creates exactly one STOPPED VM per sub-case and deletes it. Cleanup is
// registered BEFORE the create and is name-based, so it runs even on a Fatal.
func TestVPCVMCreateLiveProbePhase2VMI(t *testing.T) {
	c, cfg := requireVPCCreateProbeClient(t)
	if os.Getenv("CT_VPC_VMCREATE_PROBE_WRITE") != "1" {
		t.Skip("set CT_VPC_VMCREATE_PROBE_WRITE=1 to run the controlled VM create/delete on the DEV broker")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	if _, err := c.Token(ctx); err != nil {
		t.Fatalf("auth failed: %v", err)
	}
	t.Logf("AUTH ok on %s — controlled create/delete authorised", cfg.Address)

	waitOpts := &WaiterOptions{Logger: func(m string) { t.Logf("  activity: %s", m) }}
	target := pickVMIVPCTarget(ctx, t, c)

	// Resolve a SCHEDULABLE substrate. Two live measurements shaped this:
	//
	//   1. `compatibleTargets` over-promises. Debian 12 advertises all four
	//      (family, AZ) pairs, yet creating on (Development, fr1-az01) failed with
	//      "No deployment target in availability zone '…' for instance family '…'
	//      can satisfy the requested storage layout". The authoritative feasibility
	//      signal is the STORAGE-TYPE catalogue for that exact pair.
	//   2. The image constrains the NIC COUNT: an OPNsense substrate failed with
	//      "Number of network interfaces provided (1) must be at least the image
	//      requirement (2)". The probe attaches exactly one adapter, so only an
	//      image declaring exactly one `networkInterfaces` entry is usable.
	//
	// Getting this right matters beyond convenience: a substrate failure is
	// indistinguishable from a VPC refusal, and reporting one as the other is
	// exactly the false negative this probe exists to avoid.
	imageID, azID, familyID, diskGb := resolveSchedulableVMISubstrate(ctx, t, c)
	if imageID == "" {
		t.Skip("no (image, instance family, availability zone) triple with an available storage type; cannot attribute a create failure to VPC")
	}
	policies, err := c.PublicCloudVM().BackupPolicy().List(ctx)
	if err != nil || len(policies) == 0 || policies[0] == nil {
		t.Skipf("no backup policy available (create requires a valid backupPolicyId): err=%v", err)
	}
	backupPolicyID := policies[0].ID
	cpu, ram := 1, 2
	t.Logf("SUBSTRATE image=%s az=%s family=%s systemDisk=%dGB cpu=%d ram=%d backupPolicy=%s",
		imageID, azID, familyID, diskGb, cpu, ram, backupPolicyID)

	// substrateFailure recognises a placement, quota or image-requirement refusal.
	// None of those say anything about VPC, and reporting one as a VPC verdict
	// would be a false negative — the exact mistake this probe exists to avoid.
	substrateFailure := func(msg string) bool {
		for _, marker := range []string{
			"No deployment target", "storage layout", "quota", "Quota", "insufficient",
			"image requirement", "network interfaces provided",
		} {
			if strings.Contains(msg, marker) {
				return true
			}
		}
		return false
	}

	// Cleanup BEFORE any create: name-based VM removal plus an IPAM sweep of the
	// registrations that appear on the target private network during the probe.
	// The snapshot is taken FIRST so the sweep can attribute a leak.
	staticIPsBefore := snapshotStaticIPs(ctx, t, c, target.PrivateNetworkID)
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 8*time.Minute)
		defer ccancel()
		leftovers, lerr := c.PublicCloudVM().Instance().List(cctx, &PublicCloudVMInstanceFilter{Name: vpcCreateProbeVMName})
		if lerr != nil {
			t.Logf("CLEANUP list-by-name failed: %v -- MANUAL CLEANUP MAY BE NEEDED for %q", lerr, vpcCreateProbeVMName)
		}
		for _, vm := range leftovers {
			if vm == nil || vm.Name != vpcCreateProbeVMName {
				continue
			}
			loc, derr := c.PublicCloudVM().Instance().Delete(cctx, vm.ID)
			if derr != nil {
				t.Logf("CLEANUP delete %s failed: %v -- MANUAL CLEANUP MAY BE NEEDED", vm.ID, derr)
				continue
			}
			if _, werr := c.Activity().WaitForCompletion(cctx, loc, waitOpts); werr != nil {
				t.Logf("CLEANUP delete activity for %s did not complete: %v -- MANUAL CLEANUP MAY BE NEEDED", vm.ID, werr)
				continue
			}
			t.Logf("CLEANUP deleted leftover VM %s", vm.ID)
		}
		// IPAM sweep by snapshot difference. A per-address sweep would be blind to
		// the auto-assign case (the probe never learns the address) and to a create
		// activity that registered the IP and then failed before the VM became
		// listable.
		sweepProbeStaticIPs(cctx, t, c, staticIPsBefore)
	})

	// runCase creates one stopped VM, inspects the resulting adapter and the VPC
	// static-IP registration, then deletes the VM and re-checks the registration.
	// netID selects the network kind under test (VPC target, or a Private Backbone
	// one for the "ip_address on a non-VPC network" case).
	runCaseOn := func(t *testing.T, label, netID, wantIP string, expectVPC bool) {
		nic := CreateVMInstanceNIC{DeviceIndex: 0, NetworkID: netID}
		if wantIP != "" {
			nic.IPAddress = wantIP
		}
		t.Logf("[%s] CREATE name=%s networkId=%s (expectVPC=%t) ipAddress=%q powerState=off", label, vpcCreateProbeVMName, netID, expectVPC, wantIP)

		act, err := c.PublicCloudVM().Instance().Create(ctx, &CreateVMInstanceRequest{
			Name:               vpcCreateProbeVMName,
			AvailabilityZoneID: azID,
			ImageID:            imageID,
			InstanceFamilyID:   familyID,
			CPU:                cpu,
			Memory:             ram,
			BackupPolicyID:     backupPolicyID,
			NetworkInterfaces:  []CreateVMInstanceNIC{nic},
			PowerState:         "off",
		})
		if err != nil {
			if substrateFailure(err.Error()) {
				t.Skipf("[%s] INCONCLUSIVE — the create POST failed for a placement/quota reason, not VPC: %v", label, err)
			}
			t.Fatalf("[%s] Q1 ANSWER=NO — the create POST itself was refused on a VPC network: %v", label, err)
		}
		if _, err := c.Activity().WaitForCompletion(ctx, act, waitOpts); err != nil {
			if substrateFailure(err.Error()) {
				t.Skipf("[%s] INCONCLUSIVE — the create activity failed for a placement/quota reason, not VPC: %v", label, err)
			}
			t.Fatalf("[%s] Q1 ANSWER=NO — the create ACTIVITY failed on a VPC network: %v", label, err)
		}
		if expectVPC {
			t.Logf("[%s] Q1 ANSWER=YES — a VPC networkId is accepted by the VM-create endpoint (activity completed)", label)
		} else {
			t.Logf("[%s] create accepted on a Private Backbone network (control case, not a VPC verdict)", label)
		}

		created, err := c.PublicCloudVM().Instance().List(ctx, &PublicCloudVMInstanceFilter{Name: vpcCreateProbeVMName})
		if err != nil {
			t.Fatalf("[%s] list-by-name after create failed: %v", label, err)
		}
		var vmID string
		for _, vm := range created {
			if vm != nil && vm.Name == vpcCreateProbeVMName {
				vmID = vm.ID
				break
			}
		}
		if vmID == "" {
			t.Fatalf("[%s] created VM %q not found by name after the activity completed", label, vpcCreateProbeVMName)
		}

		nics, err := c.PublicCloudVM().NetworkAdapter().List(ctx, vmID)
		if err != nil {
			t.Fatalf("[%s] NetworkAdapter().List(%s) failed: %v", label, vmID, err)
		}
		if len(nics) == 0 || nics[0] == nil {
			t.Fatalf("[%s] VM %s has no network adapter after create", label, vmID)
		}
		a := nics[0]
		// Ownership evidence for the IPAM sweep, recorded before anything can fail.
		staticIPsBefore.own(a.MacAddress)
		t.Logf("[%s] ADAPTER id=%s deviceIndex=%d networkId=%s networkName=%q type=%q provisionStatus=%q mac=%q ipv4=%q",
			label, a.ID, a.DeviceIndex, a.NetworkID, a.NetworkName, a.Type, a.ProvisionStatus, a.MacAddress, a.IPv4Address)
		wantType := "private_backbone"
		if expectVPC {
			wantType = "vpc"
		}
		if a.Type != wantType {
			t.Errorf("[%s] adapter type=%q, want %q — the VM did NOT land on the expected network kind", label, a.Type, wantType)
		}

		// The registered static IP is addressable only by MAC (#1854).
		var registered string
		if a.MacAddress != "" {
			sip, serr := c.VPC().StaticIP().ReadByMAC(ctx, a.MacAddress)
			switch {
			case serr != nil:
				t.Errorf("[%s] ReadByMAC(%s) errored: %v", label, a.MacAddress, serr)
			case sip == nil:
				t.Logf("[%s] ReadByMAC(%s) -> no registration", label, a.MacAddress)
			default:
				registered = sip.IPAddress
				t.Logf("[%s] STATIC IP registered: id=%s ip=%s source=%q privateNetwork=%s", label, sip.ID, sip.IPAddress, sip.Source, sip.PrivateNetwork.ID)
			}
		}
		switch {
		case !expectVPC && wantIP != "" && registered == "":
			t.Logf("[%s] Q7 ANSWER=SILENTLY-IGNORED — ipAddress %s was requested on a PRIVATE BACKBONE network and NO static IP was registered. The platform neither honours nor rejects it, so the provider must reject it itself or the value is dead config.", label, wantIP)
		case !expectVPC && wantIP != "":
			t.Errorf("[%s] Q7 UNEXPECTED — ipAddress %s on a non-VPC network produced a registration %s", label, wantIP, registered)
		case wantIP != "" && registered == wantIP:
			t.Logf("[%s] Q2 ANSWER=YES — networkInterfaces[].ipAddress IS honoured: requested %s, registered %s", label, wantIP, registered)
		case wantIP != "" && registered == "":
			t.Errorf("[%s] Q2 ANSWER=NO (SILENT DROP) — requested ipAddress %s but NO static IP is registered for mac %s", label, wantIP, a.MacAddress)
		case wantIP != "":
			t.Errorf("[%s] Q2 ANSWER=NO (IGNORED) — requested ipAddress %s but the registration is %s", label, wantIP, registered)
		case registered != "":
			t.Logf("[%s] Q3 ANSWER=YES — the platform auto-assigned %s with no ipAddress requested", label, registered)
		default:
			t.Logf("[%s] Q3 ANSWER=NO — no static IP is registered when ipAddress is omitted", label)
		}

		// Delete and prove the 0-orphan contract on BOTH planes.
		delAct, err := c.PublicCloudVM().Instance().Delete(ctx, vmID)
		if err != nil {
			t.Fatalf("[%s] Delete(%s) failed: %v", label, vmID, err)
		}
		if _, err := c.Activity().WaitForCompletion(ctx, delAct, waitOpts); err != nil {
			t.Fatalf("[%s] delete activity failed: %v", label, err)
		}
		gone, err := c.PublicCloudVM().Instance().Read(ctx, vmID)
		if err != nil {
			t.Fatalf("[%s] post-delete Read errored (want (nil,nil)): %v", label, err)
		}
		if gone != nil {
			t.Fatalf("[%s] VM %s still present after delete", label, vmID)
		}
		t.Logf("[%s] VM %s deleted and confirmed absent", label, vmID)

		if a.MacAddress != "" {
			sip, serr := c.VPC().StaticIP().ReadByMAC(ctx, a.MacAddress)
			switch {
			case serr != nil:
				t.Errorf("[%s] Q4 post-delete ReadByMAC errored: %v", label, serr)
			case sip == nil:
				t.Logf("[%s] Q4 ANSWER=RECLAIMED — the static-IP registration is gone after the VM delete (0-orphan on the IPAM plane)", label)
			default:
				t.Errorf("[%s] Q4 ANSWER=LEAK — static IP %s (id=%s) is STILL registered after the VM was deleted; the cleanup hook will remove it", label, sip.IPAddress, sip.ID)
			}
		}
	}

	t.Run("vpc_with_chosen_ip", func(t *testing.T) {
		runCaseOn(t, "with-ip", target.NetworkID, target.FreeIP, true)
	})
	t.Run("vpc_without_ip", func(t *testing.T) {
		runCaseOn(t, "no-ip", target.NetworkID, "", true)
	})
	// Q7: is ip_address on a NON-VPC network silently ignored, or rejected? The
	// answer decides whether the provider must reject it up front (fail closed) or
	// merely warn. A silently-ignored value is dead config the user believes took
	// effect, which argues for a hard rejection.
	t.Run("private_backbone_with_ip", func(t *testing.T) {
		pbNet := ""
		nets, err := c.PublicCloudVM().Network().List(ctx)
		if err != nil {
			t.Skipf("cannot list networks to find a Private Backbone one: %v", err)
		}
		for _, n := range nets {
			if n != nil && n.VPC == nil {
				pbNet = n.ID
				break
			}
		}
		if pbNet == "" {
			t.Skip("no Private Backbone network in the catalogue")
		}
		runCaseOn(t, "pb-with-ip", pbNet, target.FreeIP, false)
	})

	t.Log("PHASE 2 VMI DONE")
}

// resolveSchedulableVMISubstrate returns an (image, availabilityZone,
// instanceFamily, systemDiskGb) tuple that the platform can actually schedule
// with ONE network adapter, or a zero image id when none exists.
//
// It is read-only. The feasibility test is the storage-type catalogue for the
// exact (availabilityZoneId, instanceFamilyId) pair — the only signal found to
// agree with what the create activity accepts (see the caller's comment).
func resolveSchedulableVMISubstrate(ctx context.Context, t *testing.T, c *Client) (imageID, azID, familyID string, diskGb int) {
	t.Helper()

	r := c.newRequest("GET", "/vm_instances/v1/images")
	resp, err := c.doRequest(ctx, r)
	if err != nil {
		t.Fatalf("GET /images failed: %v", err)
	}
	raw, _ := io.ReadAll(resp.Body)
	closeResponseBody(resp)

	var images []struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		DiskSizesGb       []int  `json:"diskSizesGb"`
		NetworkInterfaces []struct {
			Position int `json:"position"`
		} `json:"networkInterfaces"`
		CompatibleTargets []struct {
			InstanceFamilyID   string `json:"instanceFamilyId"`
			AvailabilityZoneID string `json:"availabilityZoneId"`
		} `json:"compatibleTargets"`
	}
	if err := json.Unmarshal(raw, &images); err != nil {
		t.Fatalf("decoding /images failed: %v", err)
	}

	// Cache the storage-type lookup: the same (az, family) pair recurs across images.
	type pair struct{ az, family string }
	feasible := map[pair][]*PublicCloudVMStorageType{}
	storageTypesFor := func(p pair) []*PublicCloudVMStorageType {
		if st, ok := feasible[p]; ok {
			return st
		}
		st, err := c.PublicCloudVM().StorageType().List(ctx, &PublicCloudVMStorageTypeFilter{
			AvailabilityZoneID: p.az, InstanceFamilyID: p.family,
		})
		if err != nil {
			t.Logf("  storage_types(az=%s family=%s) failed: %v", p.az, p.family, err)
			st = nil
		}
		feasible[p] = st
		return st
	}

	for _, img := range images {
		if len(img.NetworkInterfaces) != 1 || len(img.DiskSizesGb) == 0 {
			continue
		}
		want := img.DiskSizesGb[0]
		for _, tgt := range img.CompatibleTargets {
			p := pair{az: tgt.AvailabilityZoneID, family: tgt.InstanceFamilyID}
			for _, st := range storageTypesFor(p) {
				if st == nil || !st.IsAvailable {
					continue
				}
				if want < st.MinSizeGb || (st.MaxSizeGb > 0 && want > st.MaxSizeGb) {
					continue
				}
				t.Logf("SUBSTRATE feasible: image %q (%s, 1 NIC, %dGB) on az=%s family=%s via storage type %q (%s, %d-%dGB)",
					img.Name, img.ID, want, p.az, p.family, st.Name, st.ID, st.MinSizeGb, st.MaxSizeGb)
				return img.ID, p.az, p.family, want
			}
		}
	}
	return "", "", "", 0
}

// TestVPCVMCreateLiveProbePhase2OpenIaaS is the DECISIVE experiment for the
// OpenIaaS surface. Issue #376 is Blocked on the claim that the VM-create
// endpoint's networkAdapters[] accepts only {networkId, mac} — no ipAddress.
// That claim is a reading of the broker's schema file, not live evidence, and
// the differential schema probe above proved inconclusive (the endpoint does not
// validate the body synchronously). So this creates a real VM.
//
//	Q5 does POST /compute/v1/open_iaas/virtual_machines accept a VPC networkId in
//	   networkAdapters[] at all?
//	Q6 is an ipAddress supplied there HONOURED, IGNORED (auto-assigned instead),
//	   or SILENTLY DROPPED (no registration)?
//
// The request is built as a raw map, not through CreateOpenIaasVirtualMachineRequest,
// precisely because the client struct has no IPAddress field yet: sending it
// through the typed struct would require changing production code to test an
// unknown. One VM, created and deleted, cleanup registered before the create.
func TestVPCVMCreateLiveProbePhase2OpenIaaS(t *testing.T) {
	c, cfg := requireVPCCreateProbeClient(t)
	if os.Getenv("CT_VPC_VMCREATE_PROBE_WRITE") != "1" {
		t.Skip("set CT_VPC_VMCREATE_PROBE_WRITE=1 to run the controlled VM create/delete on the DEV broker")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if _, err := c.Token(ctx); err != nil {
		t.Fatalf("auth failed: %v", err)
	}
	t.Logf("AUTH ok on %s — controlled OpenIaaS create/delete authorised", cfg.Address)
	waitOpts := &WaiterOptions{Logger: func(m string) { t.Logf("  activity: %s", m) }}

	// A template with exactly ONE network adapter, plus a VPC-backed network on
	// the SAME machine manager (an OpenIaaS network is scoped to its pool, so a
	// cross-manager network id would fail for an unrelated reason).
	templates, err := c.Compute().OpenIaaS().Template().List(ctx, nil)
	if err != nil {
		t.Fatalf("Template().List failed: %v", err)
	}
	nets, err := c.Compute().OpenIaaS().Network().List(ctx, nil)
	if err != nil {
		t.Fatalf("Network().List failed: %v", err)
	}
	// Measured on DEV: the three OpenIaaS templates declare ZERO network adapters,
	// so "exactly one" finds nothing. At most one is accepted, and the probe still
	// sends exactly one networkAdapters[] entry — which additionally measures what
	// the API does when the request declares more adapters than the template.
	// (The provider's own inline block refuses that mismatch up front:
	// resource_compute_iaas_opensource_virtual_machine.go:560-562.)
	var tmpl *OpenIaasTemplate
	var vpcNet *OpenIaaSNetwork
	for _, tp := range templates {
		if tp == nil || len(tp.NetworkAdapters) > 1 || tp.CPU <= 0 || tp.Memory <= 0 {
			continue
		}
		for _, n := range nets {
			if n == nil || n.VPC == nil || n.MachineManager.ID != tp.MachineManager.ID {
				continue
			}
			tmpl, vpcNet = tp, n
			break
		}
		if tmpl != nil {
			break
		}
	}
	if tmpl == nil {
		t.Skip("no single-adapter OpenIaaS template with a VPC-backed network on the same machine manager; cannot run the OpenIaaS write probe")
	}
	t.Logf("TEMPLATE %s (%q) machineManager=%s cpu=%d memoryBytes=%d adapters=%d",
		tmpl.ID, tmpl.Name, tmpl.MachineManager.ID, tmpl.CPU, tmpl.Memory, len(tmpl.NetworkAdapters))
	t.Logf("TARGET   VPC network %s (%q) privateNetwork=%s (%q)",
		vpcNet.ID, vpcNet.Name, vpcNet.VPC.PrivateNetwork.ID, vpcNet.VPC.PrivateNetwork.Name)

	// Choose a free address in the target private network.
	taken := map[string]bool{}
	if ips, ierr := c.VPC().StaticIP().List(ctx, vpcNet.VPC.PrivateNetwork.ID, nil); ierr == nil {
		for _, ip := range ips {
			if ip != nil {
				taken[ip.IPAddress] = true
			}
		}
	} else {
		t.Logf("StaticIP().List failed (continuing, collision risk): %v", ierr)
	}
	var cidr string
	if pns, perr := c.VPC().PrivateNetwork().List(ctx, nil); perr == nil {
		for _, p := range pns {
			if p != nil && p.ID == vpcNet.VPC.PrivateNetwork.ID {
				cidr = p.IPAddress
			}
		}
	}
	wantIP := firstFreeHostInSlash24(cidr, taken)
	if wantIP == "" {
		t.Skipf("target private network %s has CIDR %q — no /24 host range to pick a free address from", vpcNet.VPC.PrivateNetwork.ID, cidr)
	}
	t.Logf("TARGET   cidr=%s registeredStaticIPs=%d chosenFreeIP=%s", cidr, len(taken), wantIP)

	// Cleanup BEFORE the create, including the IPAM sweep the first version of
	// this probe was missing.
	staticIPsBefore := snapshotStaticIPs(ctx, t, c, vpcNet.VPC.PrivateNetwork.ID)
	cleanup := func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer ccancel()
		vms, lerr := c.Compute().OpenIaaS().VirtualMachine().List(cctx, &OpenIaaSVirtualMachineFilter{})
		if lerr != nil {
			// Do NOT return: the IPAM sweep below is independent of the VM listing
			// and must still run.
			t.Logf("CLEANUP OpenIaaS list failed: %v -- MANUAL CLEANUP MAY BE NEEDED for %q", lerr, vpcCreateProbeVMName)
		}
		for _, vm := range vms {
			if vm == nil || vm.Name != vpcCreateProbeVMName {
				continue
			}
			act, derr := c.Compute().OpenIaaS().VirtualMachine().Delete(cctx, vm.ID)
			if derr != nil {
				t.Logf("CLEANUP delete %s failed: %v -- MANUAL CLEANUP MAY BE NEEDED", vm.ID, derr)
				continue
			}
			if _, werr := c.Activity().WaitForCompletion(cctx, act, waitOpts); werr != nil {
				t.Logf("CLEANUP delete activity for %s did not complete: %v -- MANUAL CLEANUP MAY BE NEEDED", vm.ID, werr)
				continue
			}
			t.Logf("CLEANUP deleted OpenIaaS VM %s", vm.ID)
		}
		sweepProbeStaticIPs(cctx, t, c, staticIPsBefore)
	}
	t.Cleanup(cleanup)

	// Raw create carrying ipAddress alongside networkId.
	body := map[string]interface{}{
		"name":       vpcCreateProbeVMName,
		"templateId": tmpl.ID,
		"cpu":        tmpl.CPU,
		"memory":     tmpl.Memory,
		"networkAdapters": []interface{}{
			map[string]interface{}{"networkId": vpcNet.ID, "ipAddress": wantIP},
		},
	}
	t.Logf("CREATE (raw) templateId=%s networkAdapters=[{networkId:%s, ipAddress:%s}]", tmpl.ID, vpcNet.ID, wantIP)
	r := c.newRequest("POST", "/compute/v1/open_iaas/virtual_machines")
	r.obj = body
	activityID, err := c.doRequestAndReturnActivity(ctx, r)
	if err != nil {
		t.Fatalf("Q5 ANSWER=NO — the OpenIaaS create POST was refused: %v", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityID, waitOpts)
	if err != nil {
		t.Fatalf("Q5 ANSWER=NO — the OpenIaaS create ACTIVITY failed on a VPC network: %v", err)
	}
	t.Log("Q5 ANSWER=YES — a VPC networkId is accepted by the OpenIaaS VM-create endpoint")

	// The OpenIaaS VM id comes from the terminal activity state's Result (the
	// same source setIdFromActivityState uses), with a name-based fallback.
	vmID := ""
	if activity != nil {
		for _, st := range activity.State {
			if st.Result != "" {
				vmID = st.Result
			}
		}
	}
	if vmID == "" {
		vms, lerr := c.Compute().OpenIaaS().VirtualMachine().List(ctx, &OpenIaaSVirtualMachineFilter{})
		if lerr != nil {
			t.Fatalf("cannot resolve the created VM: %v", lerr)
		}
		for _, vm := range vms {
			if vm != nil && vm.Name == vpcCreateProbeVMName {
				vmID = vm.ID
			}
		}
	}
	if vmID == "" {
		t.Fatalf("created OpenIaaS VM %q not found after the activity completed", vpcCreateProbeVMName)
	}
	t.Logf("CREATED OpenIaaS VM %s", vmID)

	adapters, err := c.Compute().OpenIaaS().NetworkAdapter().List(ctx, &OpenIaaSNetworkAdapterFilter{VirtualMachineID: vmID})
	if err != nil {
		t.Fatalf("NetworkAdapter().List(%s) failed: %v", vmID, err)
	}
	if len(adapters) == 0 || adapters[0] == nil {
		t.Fatalf("OpenIaaS VM %s has no network adapter after create", vmID)
	}
	a := adapters[0]
	staticIPsBefore.own(a.MacAddress)
	onVPC := a.VPC != nil
	vpcStatic := ""
	if onVPC {
		vpcStatic = a.VPC.StaticIPAddress
	}
	t.Logf("ADAPTER id=%s network=%s(%q) mac=%s ipv4=%q onVPC=%t vpc.staticIpAddress=%q",
		a.ID, a.Network.ID, a.Network.Name, a.MacAddress, a.IPv4Address, onVPC, vpcStatic)
	if !onVPC {
		t.Errorf("the created adapter is NOT on a VPC (vpc == nil) although networkId %s is VPC-backed", vpcNet.ID)
	}

	registered := ""
	if a.MacAddress != "" {
		sip, serr := c.VPC().StaticIP().ReadByMAC(ctx, a.MacAddress)
		switch {
		case serr != nil:
			t.Errorf("ReadByMAC(%s) errored: %v", a.MacAddress, serr)
		case sip == nil:
			t.Logf("ReadByMAC(%s) -> no registration", a.MacAddress)
		default:
			registered = sip.IPAddress
			t.Logf("STATIC IP registered: id=%s ip=%s source=%q privateNetwork=%s", sip.ID, sip.IPAddress, sip.Source, sip.PrivateNetwork.ID)
		}
	}
	switch {
	case registered == wantIP:
		t.Logf("Q6 ANSWER=HONOURED — networkAdapters[].ipAddress works on the OpenIaaS VM-create path: requested %s, registered %s. Issue #376's API gap is CLOSED.", wantIP, registered)
	case registered != "":
		t.Logf("Q6 ANSWER=IGNORED — ipAddress %s was dropped and the platform auto-assigned %s instead. The inline block can attach to a VPC but cannot CHOOSE the address at create.", wantIP, registered)
	default:
		t.Logf("Q6 ANSWER=NO-REGISTRATION — no static IP is registered for mac %s at all.", a.MacAddress)
	}

	// Delete now (do not rely on the cleanup hook) and prove both planes.
	delAct, err := c.Compute().OpenIaaS().VirtualMachine().Delete(ctx, vmID)
	if err != nil {
		t.Fatalf("Delete(%s) failed: %v", vmID, err)
	}
	if _, err := c.Activity().WaitForCompletion(ctx, delAct, waitOpts); err != nil {
		t.Fatalf("delete activity failed: %v", err)
	}
	t.Logf("OpenIaaS VM %s deleted", vmID)
	if a.MacAddress != "" {
		sip, serr := c.VPC().StaticIP().ReadByMAC(ctx, a.MacAddress)
		switch {
		case serr != nil:
			t.Errorf("post-delete ReadByMAC errored: %v", serr)
		case sip == nil:
			t.Log("Q4/OI ANSWER=RECLAIMED — the static-IP registration is gone after the VM delete")
		default:
			t.Errorf("Q4/OI ANSWER=LEAK — static IP %s (id=%s) is STILL registered after the VM was deleted", sip.IPAddress, sip.ID)
		}
	}
	t.Log("PHASE 2 OPENIAAS DONE")
}

// staticIPBaseline is the pre-write picture of a private network's IPAM. `ok` is
// false when the snapshot could not be taken: the destructive sweep is then
// DISABLED entirely rather than run against an empty baseline, which would treat
// every pre-existing registration as newly created.
//
// ownedMACs accumulates the MAC addresses of the adapters the probe's own VMs came
// up with. It is the ONLY strict positive ownership evidence available for a static
// IP: the registration is keyed by MAC on the VPC plane, and a MAC belongs to the
// machine the probe created. The map is shared with the cleanup closure on purpose,
// so a MAC observed mid-run is visible to a sweep that runs later.
type staticIPBaseline struct {
	privateNetworkID string
	addresses        map[string]bool
	ownedMACs        map[string]bool
	ok               bool
}

// own records a MAC the probe's own VM came up with, normalised for comparison.
func (b *staticIPBaseline) own(mac string) {
	if mac != "" {
		b.ownedMACs[strings.ToLower(mac)] = true
	}
}

// snapshotStaticIPs records the static-IP addresses registered on a private
// network BEFORE a probe write, so the cleanup can tell a leak the probe caused
// apart from a registration that already existed (or that another operator
// created meanwhile).
func snapshotStaticIPs(ctx context.Context, t *testing.T, c *Client, privateNetworkID string) staticIPBaseline {
	t.Helper()
	base := staticIPBaseline{privateNetworkID: privateNetworkID, addresses: map[string]bool{}, ownedMACs: map[string]bool{}}
	ips, err := c.VPC().StaticIP().List(ctx, privateNetworkID, nil)
	if err != nil {
		t.Logf("SNAPSHOT static-IP list of %s failed: %v -- the destructive IPAM sweep is DISABLED for this run; any leak will be reported for MANUAL CLEANUP instead", privateNetworkID, err)
		return base
	}
	for _, ip := range ips {
		if ip != nil {
			base.addresses[ip.IPAddress] = true
		}
	}
	base.ok = true
	return base
}

// sweepProbeStaticIPs reports the static-IP registrations that appeared on the
// private network since the baseline, and deletes ONLY those the probe can
// positively prove it owns:
//
//   - the registration is attached to a VM carrying the probe's exact name, or
//   - its MAC address is one the probe's own VM came up with (base.ownedMACs).
//
// Address equality is deliberately NOT ownership evidence. The probe picks a
// currently-free address, but on a shared tenant another operator can register
// that same address between the baseline and the sweep; deleting on equality alone
// would destroy their live registration. Likewise an UNATTACHED registration is not
// evidence — a legitimate one can be unattached at that instant.
//
// Anything the probe cannot prove it owns is logged for manual cleanup and left
// alone. That is a deliberate trade: .clinerules/WORKFLOW_ENGINEERING.md §5 requires
// strict positive evidence for a destructive decision, so a possible manual cleanup
// is the correct cost. The whole destructive path is skipped when the baseline could
// not be taken at all.
func sweepProbeStaticIPs(ctx context.Context, t *testing.T, c *Client, base staticIPBaseline) {
	t.Helper()
	ips, err := c.VPC().StaticIP().List(ctx, base.privateNetworkID, nil)
	if err != nil {
		t.Logf("CLEANUP static-IP list of %s failed: %v -- MANUAL CLEANUP NEEDED (IPAM audit)", base.privateNetworkID, err)
		return
	}
	for _, ip := range ips {
		if ip == nil || base.addresses[ip.IPAddress] {
			continue
		}
		owner := "<unattached>"
		if ip.VirtualMachine != nil {
			owner = ip.VirtualMachine.Name
		}
		if !base.ok {
			t.Logf("CLEANUP static IP %s (id=%s, owner=%s) is new relative to an UNTAKEN baseline -- NOT deleting (no reliable evidence); MANUAL CLEANUP NEEDED", ip.IPAddress, ip.ID, owner)
			continue
		}
		ownedByProbe := (ip.VirtualMachine != nil && ip.VirtualMachine.Name == vpcCreateProbeVMName) ||
			base.ownedMACs[strings.ToLower(ip.MacAddress)]
		if !ownedByProbe {
			t.Logf("CLEANUP static IP %s (id=%s, source=%q, mac=%q, owner=%s) appeared during the probe but ownership is NOT provable -- NOT deleting; MANUAL CLEANUP NEEDED if this was a probe leak", ip.IPAddress, ip.ID, ip.Source, ip.MacAddress, owner)
			continue
		}
		t.Logf("CLEANUP LEAK: static IP %s (id=%s, source=%q, mac=%q, owner=%s) is provably this probe's -- deleting", ip.IPAddress, ip.ID, ip.Source, ip.MacAddress, owner)
		if _, derr := c.VPC().StaticIP().Delete(ctx, ip.ID); derr != nil {
			t.Logf("CLEANUP static-IP delete %s failed: %v -- MANUAL CLEANUP NEEDED", ip.ID, derr)
		}
	}
}

// TestVPCVMwareAdapterLiveProbe is READ-ONLY and covers the one contract the
// VMware inline path newly depends on and that no other probe here establishes:
// that a vCenter network ADAPTER sitting on a VPC-backed network really does expose
// the `vpc` object, and that its registered static IP resolves by MAC.
//
// Why read-only here rather than a create/delete cycle: the full VMware
// create-with-a-chosen-address cycle IS proven live, by
// TestAccResourceVirtualMachineVPCInline in internal/provider/tests — it clones a VM
// onto a VPC network with a chosen static IP and asserts the LIVE registration by
// MAC. This probe covers the complementary INVENTORY question cheaply, over EXISTING
// machines: do vCenter adapters on VPC networks really expose `vpc`, and does their
// address resolve by MAC? Answering that needs no creation at all.
func TestVPCVMwareAdapterLiveProbe(t *testing.T) {
	c, cfg := requireVPCCreateProbeClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	if _, err := c.Token(ctx); err != nil {
		t.Fatalf("auth failed: %v", err)
	}
	t.Logf("AUTH ok on %s — READ-ONLY VMware adapter probe", cfg.Address)

	vms, err := c.Compute().VirtualMachine().List(ctx, &VirtualMachineFilter{})
	if err != nil {
		t.Fatalf("VirtualMachine().List failed: %v", err)
	}
	t.Logf("vCenter VMs visible: %d", len(vms))

	inspected, onVPC := 0, 0
	for _, vm := range vms {
		if vm == nil {
			continue
		}
		adapters, aerr := c.Compute().NetworkAdapter().List(ctx, &NetworkAdapterFilter{VirtualMachineID: vm.ID})
		if aerr != nil {
			continue
		}
		for _, a := range adapters {
			if a == nil {
				continue
			}
			inspected++
			if a.VPC == nil {
				continue
			}
			onVPC++
			registered := "<none>"
			if a.MacAddress != "" {
				sip, serr := c.VPC().StaticIP().ReadByMAC(ctx, a.MacAddress)
				switch {
				case serr != nil:
					registered = "ERR:" + serr.Error()
				case sip != nil:
					registered = sip.IPAddress + " (source=" + sip.Source + ")"
				}
			}
			t.Logf("VW adapter on VPC: vm=%q nic=%s net=%s(%q) mac=%s autoConnect=%t connected=%t vpc={id:%s privateNetwork:%s} staticIP=%s",
				vm.Name, a.ID, a.Network.ID, a.Network.Name, a.MacAddress, a.AutoConnect, a.Connected,
				a.VPC.ID, a.VPC.PrivateNetwork.ID, registered)
			if onVPC >= 5 {
				break
			}
		}
		if onVPC >= 5 {
			break
		}
	}
	t.Logf("adapters inspected=%d, on a VPC=%d", inspected, onVPC)
	if onVPC == 0 {
		t.Skip("no vCenter adapter currently sits on a VPC-backed network; the read contract cannot be observed on this tenant right now")
	}
}
