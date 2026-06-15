package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// This file holds the PRE-CREATE teardown registration logic (F3). Every write
// cycle registers a best-effort, idempotent teardown keyed by a DETERMINISTIC
// identifier BEFORE the creating call, so a created-but-unresolved or
// created-but-failed resource is still swept. The client documents several
// "201 with empty/ambiguous body" paths where the create succeeds server-side
// but id resolution can fail (e.g. internal/client/vpc_static_ip.go) — those
// are exactly the orphan windows this pre-registration closes.
//
// Each registration takes a narrow SEAM interface (only the methods the
// teardown needs), not *client.Client, so it is unit-testable offline with a
// fake that returns an error AFTER simulating the server-side effect.
//
// Idempotency contract for every teardown here (F3): "absent = success" means a
// DEFINITIVE HTTP 404 only, never a blanket 4xx. A 403/409/400 is NOT proof of
// absence — treating it as "already gone" would let a delete/revoke/unbind
// report success while a bucket, ACL grant, FIP binding or static IP still
// exists (an orphan). So:
//   - bucket delete, ACL revoke, PAT delete, static IP delete: a 404 is success
//     (idempotent: a second delete of an already-removed resource returns 404),
//     any other error is surfaced as a real failure;
//   - the static IP delete additionally mirrors #312: a 403 is AMBIGUOUS (the
//     VPC API conflates absent/forbidden, #303), so it is confirmed via a strict
//     200-only listing of the private network before being accepted as gone;
//   - the FIP unbind mirrors the merged #309 confirmFloatingIPUnbound doctrine:
//     a 404 is success, a 403/other is NEVER assumed gone — it is positively
//     confirmed via the strict listing (CorroborateBinding), accepted only on
//     proof the pair is no longer bound (Unbound or BoundToOther).

// isStatusCode reports whether err is (or wraps) a client.StatusError with the
// given HTTP code. This is the single source of truth for the "404-only is
// absent" contract — mirrors isVPCStatusCode in the provider so the harness and
// the resource layer cannot drift on what counts as a definitive not-found.
func isStatusCode(err error, code int) bool {
	var statusErr client.StatusError
	return errors.As(err, &statusErr) && statusErr.Code == code
}

// idempotentDeleteErr is the SHARED delete/revoke idempotency decision for the
// seams whose absence is unambiguous (bucket, ACL grant, PAT): a DEFINITIVE 404
// means the resource is already gone → success (nil); ANY other error (403, 409,
// 400, 5xx, transport) is NOT proof of absence and is surfaced unchanged. Both
// the production seams and the offline unit tests call THIS function, so a
// mutation here breaks production and the tests together (no parallel copy).
func idempotentDeleteErr(err error) error {
	if err == nil {
		return nil
	}
	if isStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

// staticIPDeleteErrResult is the SHARED static-IP delete idempotency decision
// (#312): a 404 is idempotent success; a 403 is AMBIGUOUS (#303 conflates
// absent/forbidden) and is confirmed via a strict 200-only listing of the
// private network before being accepted; any other error surfaces. listStrict
// is injected so the decision is unit-testable offline.
func staticIPDeleteErrResult(err error, listStrict func() ([]*client.StaticIP, error), privateNetworkID, id string) error {
	if err == nil {
		return nil
	}
	if isStatusCode(err, http.StatusNotFound) {
		return nil
	}
	if isStatusCode(err, http.StatusForbidden) {
		if privateNetworkID == "" {
			return fmt.Errorf("static IP %s delete returned 403 and its absence could not be confirmed (no private network scope): %w", id, err)
		}
		list, lerr := listStrict()
		if lerr != nil {
			return fmt.Errorf("static IP %s delete returned 403 and the strict listing of private network %s failed: %w (original: %v)", id, privateNetworkID, lerr, err)
		}
		for _, si := range list {
			if si != nil && si.ID == id {
				return fmt.Errorf("static IP %s could not be deleted (403) and is still present on private network %s: %w", id, privateNetworkID, err)
			}
		}
		// Confirmed absent from a complete 200 listing of its own network.
		return nil
	}
	return err
}

// fipUnbindOutcome is the SHARED floating-IP unbind confirmation decision (#309
// confirmFloatingIPUnbound doctrine): the unbind is accepted ONLY on strict
// positive evidence the FIP is no longer bound to OUR static IP (Unbound or
// BoundToOther). A failed corroboration, a still-our-pair (BoundToTarget), or an
// inconclusive listing all FAIL CLOSED — there is NO "absent from listing =>
// success" path. unbindErr (when non-nil) is woven into the failure detail.
func fipUnbindOutcome(state client.FloatingIPBindingState, corrErr error, fipID, staticID string, unbindErr error) error {
	if corrErr != nil {
		return fmt.Errorf("floating IP %s unbind from %s could not be confirmed (strict listing failed): %w (original: %v)", fipID, staticID, corrErr, unbindErr)
	}
	switch state {
	case client.FloatingIPBindingUnbound, client.FloatingIPBindingBoundToOther:
		// No longer bound to OUR pair: the unbind took effect → success.
		return nil
	case client.FloatingIPBindingBoundToTarget:
		return fmt.Errorf("floating IP %s is still bound to static IP %s after the unbind (confirmed by the strict listing): %v", fipID, staticID, unbindErr)
	default:
		return fmt.Errorf("floating IP %s unbind from %s could not be positively confirmed (inconclusive listing): %v", fipID, staticID, unbindErr)
	}
}

// --- VPC static IP -----------------------------------------------------------

// staticIPSeam is the subset of the VPC static-IP client a static-IP teardown
// needs. *client.Client satisfies it via vpcStaticIPSeam.
type staticIPSeam interface {
	// ListStrict returns a provably-complete listing of the private network's
	// static IPs (fails closed otherwise — see the client doc).
	ListStrict(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error)
	// DeleteAndWait deletes a static IP by id and waits for the delete activity.
	// It is idempotent under the F3 contract: a 404 is success; a 403 is
	// confirmed absent via a strict listing of privateNetworkID before being
	// accepted (mirrors #312); any other error is surfaced.
	DeleteAndWait(ctx context.Context, privateNetworkID, id string) error
}

// registerStaticIPTeardown registers a teardown that finds the custom static IP
// carrying our MAC on the private network and deletes it if present. It is
// registered BEFORE Create, so even an ambiguous/failed create (201 empty body,
// id unresolved) is still swept by the next teardown pass. The deterministic key
// is (privateNetworkID, mac); no created id is needed.
//
// Idempotent: if no custom static IP carries our MAC, the network is already
// clean and the teardown succeeds.
func registerStaticIPTeardown(cl *Cleanup, seam staticIPSeam, privateNetworkID, mac string) {
	cl.Register(fmt.Sprintf("vpc.static_ip by-mac %s on %s", mac, privateNetworkID), func(tctx context.Context) error {
		list, err := seam.ListStrict(tctx, privateNetworkID)
		if err != nil {
			return err
		}
		want := normalizeMACForCleanup(mac)
		for _, si := range list {
			if si == nil || si.Source != "custom" {
				continue
			}
			if normalizeMACForCleanup(si.MacAddress) != want {
				continue
			}
			// Found our created-but-maybe-unresolved static IP: delete by id.
			return seam.DeleteAndWait(tctx, privateNetworkID, si.ID)
		}
		// Absent → already clean → success (idempotent).
		return nil
	})
}

// normalizeMACForCleanup canonicalises a MAC for comparison the same way the
// client does (lowercase, ":"-separated). Kept local to avoid depending on an
// unexported client helper.
func normalizeMACForCleanup(mac string) string {
	out := make([]rune, 0, len(mac))
	for _, r := range mac {
		switch {
		case r == '-':
			out = append(out, ':')
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

// vpcStaticIPSeam adapts *client.Client to staticIPSeam.
type vpcStaticIPSeam struct{ c *client.Client }

func (s vpcStaticIPSeam) ListStrict(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
	return s.c.VPC().StaticIP().ListStrict(ctx, privateNetworkID)
}

func (s vpcStaticIPSeam) DeleteAndWait(ctx context.Context, privateNetworkID, id string) error {
	activityID, err := s.c.VPC().StaticIP().Delete(ctx, id)
	if err != nil {
		// 404 → idempotent success; 403 → confirm absence via the strict listing
		// (#312); anything else surfaces. The decision lives in a shared, offline-
		// testable helper so production and tests cannot drift.
		return staticIPDeleteErrResult(err, func() ([]*client.StaticIP, error) {
			return s.ListStrict(ctx, privateNetworkID)
		}, privateNetworkID, id)
	}
	if activityID == "" {
		return nil
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr
}

// --- Object storage bucket ----------------------------------------------------

// bucketSeam is the subset of the bucket client a bucket teardown needs.
type bucketSeam interface {
	// DeleteAndWait deletes a bucket by name and waits for the delete activity.
	// It MUST treat an already-absent bucket (404) as success; any other error
	// (403/409/400/5xx/transport) is a real failure.
	DeleteAndWait(ctx context.Context, name string) error
}

// registerBucketTeardown registers "delete bucket by name if present" BEFORE the
// create, keyed by the deterministic bucket name, so a created-but-unconfirmed
// bucket is still swept. Idempotent via the seam's 404-is-success contract.
func registerBucketTeardown(cl *Cleanup, seam bucketSeam, name string) {
	cl.Register(fmt.Sprintf("object_storage.bucket by-name %s", name), func(tctx context.Context) error {
		return seam.DeleteAndWait(tctx, name)
	})
}

// objectStorageBucketSeam adapts *client.Client to bucketSeam.
type objectStorageBucketSeam struct{ c *client.Client }

func (s objectStorageBucketSeam) DeleteAndWait(ctx context.Context, name string) error {
	activityID, err := s.c.ObjectStorage().Bucket().Delete(ctx, name)
	if err != nil {
		// Only a DEFINITIVE 404 proves the bucket is gone → idempotent success.
		// A 403/409/400 is NOT proof of absence (shared decision; see helper).
		return idempotentDeleteErr(err)
	}
	if activityID == "" {
		return nil
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr
}

// --- Object storage ACL entry -------------------------------------------------

// aclSeam is the subset of the ACL-entry client an ACL teardown needs.
type aclSeam interface {
	// RevokeAndWait revokes (role, account) on the bucket and waits. It MUST
	// treat an already-absent grant (404) as success; any other error is a real
	// failure.
	RevokeAndWait(ctx context.Context, bucket, role, account string) error
}

// registerACLTeardown registers revoke(role, account) BEFORE the grant, keyed by
// the deterministic (bucket, role, account) triple, so an ambiguous grant is
// still swept. Idempotent via the seam's 404-is-success contract.
func registerACLTeardown(cl *Cleanup, seam aclSeam, bucket, role, account string) {
	cl.Register(fmt.Sprintf("object_storage.acl revoke %s/%s/%s", bucket, role, account), func(tctx context.Context) error {
		return seam.RevokeAndWait(tctx, bucket, role, account)
	})
}

// objectStorageACLSeam adapts *client.Client to aclSeam.
type objectStorageACLSeam struct{ c *client.Client }

func (s objectStorageACLSeam) RevokeAndWait(ctx context.Context, bucket, role, account string) error {
	activityID, err := s.c.ObjectStorage().ACLEntry().Revoke(ctx, bucket, role, account)
	if err != nil {
		// Only a 404 proves the grant is already gone → idempotent success. A
		// 403/409/400 must NOT be read as "already revoked" (shared decision).
		return idempotentDeleteErr(err)
	}
	if activityID == "" {
		return nil
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr
}

// --- VPC floating IP binding --------------------------------------------------

// fipBindSeam is the subset of the floating-IP client a binding teardown needs.
type fipBindSeam interface {
	// UnbindAndWait unbinds the floating IP from the static IP and waits. A 404
	// is idempotent success; a 403/other is NEVER assumed "gone" — it is
	// positively confirmed via CorroborateBinding before being accepted.
	UnbindAndWait(ctx context.Context, fipID, staticID string) error
	// CorroborateBinding strictly classifies the FIP/static relationship from a
	// COMPLETE 200 listing (fails closed to Inconclusive otherwise).
	CorroborateBinding(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error)
}

// registerFIPUnbindTeardown registers unbind(fip, static) BEFORE the bind, keyed
// by the deterministic (fipID, staticID) pair, so a bind whose confirmation is
// lost is still released. Idempotent via the seam's 404/confirmed contract.
func registerFIPUnbindTeardown(cl *Cleanup, seam fipBindSeam, fipID, staticID string) {
	cl.Register(fmt.Sprintf("vpc.floating_ip unbind %s<-%s", fipID, staticID), func(tctx context.Context) error {
		return seam.UnbindAndWait(tctx, fipID, staticID)
	})
}

// vpcFIPBindSeam adapts *client.Client to fipBindSeam.
type vpcFIPBindSeam struct{ c *client.Client }

func (s vpcFIPBindSeam) UnbindAndWait(ctx context.Context, fipID, staticID string) error {
	activityID, err := s.c.VPC().FloatingIP().Unbind(ctx, fipID, staticID)
	if err != nil {
		// 404 on the unbind call itself: unambiguous absence → idempotent success.
		if isStatusCode(err, http.StatusNotFound) {
			return nil
		}
		// 403 or any other error: NEVER assume the pair is gone (mirrors the merged
		// #309 confirmFloatingIPUnbound doctrine). Positively confirm via the strict
		// listing before accepting; an unproven state fails closed.
		state, cerr := s.CorroborateBinding(ctx, fipID, staticID)
		return fipUnbindOutcome(state, cerr, fipID, staticID, err)
	}
	if activityID != "" {
		if _, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter); werr != nil {
			return werr
		}
	}
	// Happy path too is positively confirmed: an unbind activity completing does
	// not by itself prove the pair is no longer bound.
	state, cerr := s.CorroborateBinding(ctx, fipID, staticID)
	return fipUnbindOutcome(state, cerr, fipID, staticID, nil)
}

func (s vpcFIPBindSeam) CorroborateBinding(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
	return s.c.VPC().FloatingIP().CorroborateBinding(ctx, fipID, staticID)
}

// --- IAM personal access token ------------------------------------------------

// patSeam is the subset of the PAT client a PAT teardown needs.
type patSeam interface {
	// Delete removes a PAT by id. A 404 is idempotent success; any other error is
	// a real failure.
	Delete(ctx context.Context, patID string) error
	// FindIDByName returns the id of a PAT whose name matches, or "" if none.
	// Used to remove a created-but-undecoded PAT best-effort.
	FindIDByName(ctx context.Context, name string) (string, error)
}

// patTeardownRef carries the PAT identity for teardown. The id is filled in once
// the create decodes; if it never does, the teardown falls back to find-by-name.
// A pointer is shared between the cycle and the registered closure so the cycle
// can set Resolved/ID AFTER registration without re-registering.
type patTeardownRef struct {
	Name     string
	ID       string
	Resolved bool
}

// registerPATTeardown registers a PAT teardown BEFORE the create/decode, keyed by
// the deterministic name and an id-by-reference. At teardown time it deletes by
// id when resolved, else best-effort finds the PAT by name and deletes it — so a
// created-but-undecoded PAT (a live credential) is still removed. A PAT left
// orphaned is a security issue, hence the pre-registration.
//
// Idempotent: a nil-id, name-not-found PAT means nothing to delete → success;
// a delete that 404s (already removed by the happy path) is also success.
func registerPATTeardown(cl *Cleanup, seam patSeam, ref *patTeardownRef) {
	cl.Register(fmt.Sprintf("iam.pat %s", ref.Name), func(tctx context.Context) error {
		if ref.Resolved && ref.ID != "" {
			return seam.Delete(tctx, ref.ID)
		}
		id, err := seam.FindIDByName(tctx, ref.Name)
		if err != nil {
			return err
		}
		if id == "" {
			return nil // never created / already gone → idempotent success
		}
		return seam.Delete(tctx, id)
	})
}

// iamPATSeam adapts *client.Client to patSeam.
type iamPATSeam struct{ c *client.Client }

func (s iamPATSeam) Delete(ctx context.Context, patID string) error {
	// A 404 proves the PAT is already gone → idempotent success (so the happy-
	// path-then-deferred double delete does not report a false failure). Any other
	// error (403/409/400/5xx/transport) is surfaced: a PAT is a live credential,
	// so a non-404 delete error must be reported, never swallowed (shared helper).
	return idempotentDeleteErr(s.c.IAM().PAT().Delete(ctx, patID))
}

func (s iamPATSeam) FindIDByName(ctx context.Context, name string) (string, error) {
	pats, err := s.c.IAM().PAT().ListStrict(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range pats {
		if p != nil && p.Name == name && p.ID != "" {
			return p.ID, nil
		}
	}
	return "", nil
}

// --- OpenIaaS compute lifecycle teardowns (#316 compute_lifecycle) -----------
//
// Teardown ordering is leaves-first (LIFO over registration order): the network
// adapter and the user data disk are removed BEFORE the VM anchor, because a VM
// delete does NOT cascade a user-created disk (it would orphan storage) and a
// still-connected disk can refuse deletion. Each teardown is registered BEFORE
// the create it undoes (F3), keyed by a deterministic identity, and finds its
// resource via a STRICT listing when the create's activity did not resolve the
// id. Q4/#303 note: a compute DELETE that answered 403 for an absent resource
// would surface here as a teardown FAILURE (a false alarm, never an orphan — the
// safe direction); only a definitive 404 is idempotent success.

// vmSeam is the subset of the VM client a VM teardown needs.
type vmSeam interface {
	DeleteAndWait(ctx context.Context, id string) error
	PowerOffAndWait(ctx context.Context, id string) error // best-effort, never fatal
	FindIDByName(ctx context.Context, name, machineManagerID string) (string, error)
}

// vmTeardownRef carries the VM identity; ID is filled once the create activity
// resolves it (shared pointer, like patTeardownRef). MachineManagerID scopes the
// fallback find-by-name (the OpenIaaS list 5xx's without a scope).
type vmTeardownRef struct {
	Name             string
	MachineManagerID string
	ID               string
	Resolved         bool
}

// registerVMTeardown registers the VM teardown (the anchor; runs LAST under LIFO).
// Best-effort power-off (a running VM can refuse delete) then delete; if the id
// never resolved, find the VM by its deterministic name within the machine
// manager and delete that.
func registerVMTeardown(cl *Cleanup, seam vmSeam, ref *vmTeardownRef) {
	cl.Register(fmt.Sprintf("compute.openiaas.virtual_machine %s", ref.Name), func(tctx context.Context) error {
		id := ref.ID
		if !ref.Resolved || id == "" {
			found, err := seam.FindIDByName(tctx, ref.Name, ref.MachineManagerID)
			if err != nil {
				return err
			}
			if found == "" {
				return nil // never created / already gone → idempotent success
			}
			id = found
		}
		_ = seam.PowerOffAndWait(tctx, id) // best-effort: a powered-on VM may refuse delete
		return seam.DeleteAndWait(tctx, id)
	})
}

type computeVMSeam struct{ c *client.Client }

func (s computeVMSeam) DeleteAndWait(ctx context.Context, id string) error {
	activityID, err := s.c.Compute().OpenIaaS().VirtualMachine().Delete(ctx, id)
	if err != nil {
		return idempotentDeleteErr(err) // 404 → already gone; other surfaced
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr // a failed delete activity is a real teardown failure
}

func (s computeVMSeam) PowerOffAndWait(ctx context.Context, id string) error {
	activityID, err := s.c.Compute().OpenIaaS().VirtualMachine().Power(ctx, id,
		&client.UpdateOpenIaasVirtualMachinePowerRequest{PowerState: "off", Force: true})
	if err != nil {
		return nil // best-effort; the subsequent delete surfaces a real problem
	}
	_, _ = s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return nil
}

func (s computeVMSeam) FindIDByName(ctx context.Context, name, machineManagerID string) (string, error) {
	vms, err := s.c.Compute().OpenIaaS().VirtualMachine().ListStrict(ctx,
		&client.OpenIaaSVirtualMachineFilter{MachineManagerID: machineManagerID})
	if err != nil {
		return "", err
	}
	var found string
	for _, vm := range vms {
		if vm != nil && vm.Name == name && vm.ID != "" {
			if found != "" {
				// Ambiguous: the run-unique name should match at most one. More
				// than one means an anomaly — fail closed (surface), never delete
				// a possibly-wrong VM.
				return "", fmt.Errorf("ambiguous: more than one virtual machine named %q", name)
			}
			found = vm.ID
		}
	}
	return found, nil
}

// diskSeam is the subset of the virtual-disk client a disk teardown needs.
type diskSeam interface {
	ReadConnections(ctx context.Context, id string) ([]string, error) // connected VM ids
	DisconnectAndWait(ctx context.Context, id, vmID string) error
	DeleteAndWait(ctx context.Context, id string) error
	FindIDByName(ctx context.Context, name, vmID string) (string, error)
}

type diskTeardownRef struct {
	Name     string
	VMID     string
	ID       string
	Resolved bool
}

// registerVirtualDiskTeardown registers the user data-disk teardown: disconnect
// from EVERY connected VM (a VM delete never cascades a user disk) THEN delete.
// Registered before the disk create; runs before the VM teardown (LIFO).
func registerVirtualDiskTeardown(cl *Cleanup, seam diskSeam, ref *diskTeardownRef) {
	cl.Register(fmt.Sprintf("compute.openiaas.virtual_disk %s", ref.Name), func(tctx context.Context) error {
		id := ref.ID
		if !ref.Resolved || id == "" {
			found, err := seam.FindIDByName(tctx, ref.Name, ref.VMID)
			if err != nil {
				return err
			}
			if found == "" {
				return nil
			}
			id = found
		}
		vmIDs, err := seam.ReadConnections(tctx, id)
		if err != nil {
			return err
		}
		for _, vmID := range vmIDs {
			if derr := seam.DisconnectAndWait(tctx, id, vmID); derr != nil {
				return derr
			}
		}
		return seam.DeleteAndWait(tctx, id)
	})
}

type computeVirtualDiskSeam struct{ c *client.Client }

func (s computeVirtualDiskSeam) ReadConnections(ctx context.Context, id string) ([]string, error) {
	disk, err := s.c.Compute().OpenIaaS().VirtualDisk().Read(ctx, id)
	if err != nil {
		return nil, err
	}
	if disk == nil { // 403/absent mapped to nil → nothing connected to disconnect
		return nil, nil
	}
	var vmIDs []string
	for _, vm := range disk.VirtualMachines {
		if vm.Connected && vm.ID != "" {
			vmIDs = append(vmIDs, vm.ID)
		}
	}
	return vmIDs, nil
}

func (s computeVirtualDiskSeam) DisconnectAndWait(ctx context.Context, id, vmID string) error {
	activityID, err := s.c.Compute().OpenIaaS().VirtualDisk().Disconnect(ctx, id,
		&client.OpenIaaSVirtualDiskConnectionRequest{VirtualMachineID: vmID})
	if err != nil {
		return idempotentDeleteErr(err) // already disconnected/gone → ok
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr
}

func (s computeVirtualDiskSeam) DeleteAndWait(ctx context.Context, id string) error {
	activityID, err := s.c.Compute().OpenIaaS().VirtualDisk().Delete(ctx, id)
	if err != nil {
		return idempotentDeleteErr(err)
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr
}

func (s computeVirtualDiskSeam) FindIDByName(ctx context.Context, name, vmID string) (string, error) {
	// VM-scoped first, then tenant-wide (mirrors the #325 provider doctrine): a
	// created-but-UNATTACHED disk (the create materialized it but lost the result
	// before attachment) is NOT in the VM-scoped listing, so a VM-scoped-only
	// search would orphan it after the VM delete. The disk name is run-unique, so
	// a tenant-wide match is unambiguous; >1 fails closed.
	for _, filter := range []*client.OpenIaaSVirtualDiskFilter{{VirtualMachineID: vmID}, {}} {
		id, err := s.findDiskByName(ctx, name, filter)
		if err != nil {
			return "", err
		}
		if id != "" {
			return id, nil
		}
	}
	return "", nil
}

func (s computeVirtualDiskSeam) findDiskByName(ctx context.Context, name string, filter *client.OpenIaaSVirtualDiskFilter) (string, error) {
	disks, err := s.c.Compute().OpenIaaS().VirtualDisk().ListStrict(ctx, filter)
	if err != nil {
		return "", err
	}
	var found string
	for _, d := range disks {
		if d != nil && d.Name == name && d.ID != "" {
			if found != "" {
				return "", fmt.Errorf("ambiguous: more than one virtual disk named %q", name)
			}
			found = d.ID
		}
	}
	return found, nil
}

// adapterSeam is the subset of the network-adapter client an adapter teardown needs.
type adapterSeam interface {
	// FindIDsByVM returns the ids of EVERY adapter on the VM. For a
	// created-but-unresolved adapter the teardown deletes all of them: the VM is
	// run-unique and ours and is being destroyed, so removing its adapters
	// (including any template-provided one) is safe, and there is no MAC to match
	// on (the platform assigns it).
	FindIDsByVM(ctx context.Context, vmID string) ([]string, error)
	DeleteAndWait(ctx context.Context, id string) error
}

type adapterTeardownRef struct {
	VMID     string
	ID       string
	Resolved bool
}

// registerNetworkAdapterTeardown registers the adapter teardown (a network leaf;
// runs FIRST under LIFO). When the id resolved, delete it; otherwise delete every
// adapter on the (run-unique, ours) VM.
func registerNetworkAdapterTeardown(cl *Cleanup, seam adapterSeam, ref *adapterTeardownRef) {
	cl.Register(fmt.Sprintf("compute.openiaas.network_adapter %s", ref.VMID), func(tctx context.Context) error {
		if ref.Resolved && ref.ID != "" {
			return seam.DeleteAndWait(tctx, ref.ID)
		}
		ids, err := seam.FindIDsByVM(tctx, ref.VMID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if derr := seam.DeleteAndWait(tctx, id); derr != nil {
				return derr
			}
		}
		return nil
	})
}

type computeNetworkAdapterSeam struct{ c *client.Client }

func (s computeNetworkAdapterSeam) FindIDsByVM(ctx context.Context, vmID string) ([]string, error) {
	adapters, err := s.c.Compute().OpenIaaS().NetworkAdapter().ListStrict(ctx,
		&client.OpenIaaSNetworkAdapterFilter{VirtualMachineID: vmID})
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, a := range adapters {
		if a != nil && a.ID != "" {
			ids = append(ids, a.ID)
		}
	}
	return ids, nil
}

func (s computeNetworkAdapterSeam) DeleteAndWait(ctx context.Context, id string) error {
	activityID, err := s.c.Compute().OpenIaaS().NetworkAdapter().Delete(ctx, id)
	if err != nil {
		return idempotentDeleteErr(err)
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr
}
