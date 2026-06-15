package main

import (
	"context"
	"fmt"

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
// Idempotency contract for every teardown here: an absent resource is SUCCESS
// (nil error), never a failure — so the safety net does not generate noise when
// the happy-path delete already removed the resource.

// --- VPC static IP -----------------------------------------------------------

// staticIPSeam is the subset of the VPC static-IP client a static-IP teardown
// needs. *client.Client satisfies it via vpcStaticIPSeam.
type staticIPSeam interface {
	// ListStrict returns a provably-complete listing of the private network's
	// static IPs (fails closed otherwise — see the client doc).
	ListStrict(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error)
	// DeleteAndWait deletes a static IP by id and waits for the delete activity.
	DeleteAndWait(ctx context.Context, id string) error
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
			return seam.DeleteAndWait(tctx, si.ID)
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

func (s vpcStaticIPSeam) DeleteAndWait(ctx context.Context, id string) error {
	activityID, err := s.c.VPC().StaticIP().Delete(ctx, id)
	if err != nil {
		return err
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
	// It MUST treat an already-absent bucket as success.
	DeleteAndWait(ctx context.Context, name string) error
}

// registerBucketTeardown registers "delete bucket by name if present" BEFORE the
// create, keyed by the deterministic bucket name, so a created-but-unconfirmed
// bucket is still swept. Idempotent via the seam's absent-is-success contract.
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
		// A 404/absent bucket is not an orphan: swallow not-found so the safety
		// net stays idempotent. Any other error is surfaced for the retry/report.
		if categorize(err) == CategoryHTTP4xx {
			return nil
		}
		return err
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
	// treat an already-absent grant as success.
	RevokeAndWait(ctx context.Context, bucket, role, account string) error
}

// registerACLTeardown registers revoke(role, account) BEFORE the grant, keyed by
// the deterministic (bucket, role, account) triple, so an ambiguous grant is
// still swept. Idempotent via the seam's absent-is-success contract.
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
		if categorize(err) == CategoryHTTP4xx {
			return nil // absent grant → already clean
		}
		return err
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
	// UnbindAndWait unbinds the floating IP from the static IP and waits. It MUST
	// treat an already-unbound pair as success.
	UnbindAndWait(ctx context.Context, fipID, staticID string) error
}

// registerFIPUnbindTeardown registers unbind(fip, static) BEFORE the bind, keyed
// by the deterministic (fipID, staticID) pair, so a bind whose confirmation is
// lost is still released. Idempotent via the seam's absent-is-success contract.
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
		if categorize(err) == CategoryHTTP4xx {
			return nil // already unbound → idempotent success
		}
		return err
	}
	if activityID == "" {
		return nil
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr
}

// --- IAM personal access token ------------------------------------------------

// patSeam is the subset of the PAT client a PAT teardown needs.
type patSeam interface {
	// Delete removes a PAT by id (idempotent: absent → success).
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
// Idempotent: a nil-id, name-not-found PAT means nothing to delete → success.
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
	return s.c.IAM().PAT().Delete(ctx, patID)
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
