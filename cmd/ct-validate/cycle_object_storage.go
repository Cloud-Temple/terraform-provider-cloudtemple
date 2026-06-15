package main

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// objectStorageCycle drives a bucket lifecycle:
//
//	create bucket -> (if a role + storage account exist) grant an ACL entry ->
//	read the ACL entries -> revoke -> delete bucket
//
// The bucket teardown is registered the moment the create activity completes,
// so an abort still removes it. The ACL grant teardown is registered before
// the grant is corroborated, mirroring the VPC cycle's discipline.
type objectStorageCycle struct{}

func (objectStorageCycle) Name() string { return "object_storage" }
func (objectStorageCycle) Kind() Kind   { return KindWrite }

func (oc objectStorageCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	// Bucket names are globally constrained; keep them short, lowercase and
	// unique per (iteration, worker).
	name := fmt.Sprintf("ct-validate-%d-%d", r.Iteration, r.Worker)

	created := false
	if err := r.op(oc, "object_storage.bucket.create", func() error {
		activityID, cerr := c.ObjectStorage().Bucket().Create(ctx, &client.CreateBucketRequest{
			Name:       name,
			AccessType: "private",
		})
		if cerr != nil {
			return cerr
		}
		if activityID != "" {
			if _, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter); werr != nil {
				return werr
			}
		}
		created = true
		return nil
	}); err != nil || !created {
		return err
	}

	// Register bucket teardown immediately.
	r.Cleanup.Register(fmt.Sprintf("object_storage.bucket %s", name), func(tctx context.Context) error {
		activityID, derr := c.ObjectStorage().Bucket().Delete(tctx, name)
		if derr != nil {
			return derr
		}
		if activityID == "" {
			return nil
		}
		_, werr := c.Activity().WaitForCompletion(tctx, activityID, silentWaiter)
		return werr
	})

	oc.aclSubCycle(ctx, c, r, name)

	// Explicit delete on the happy path (overlaps the registered teardown by
	// design).
	_ = r.op(oc, "object_storage.bucket.delete", func() error {
		activityID, derr := c.ObjectStorage().Bucket().Delete(ctx, name)
		if derr != nil {
			return derr
		}
		if activityID == "" {
			return nil
		}
		_, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
		return werr
	})
	return nil
}

// aclSubCycle grants an ACL entry (a role to a storage account on the bucket),
// reads it back, then revokes it. When no role or no storage account exists,
// the grant/read/revoke steps are recorded as skipped.
func (oc objectStorageCycle) aclSubCycle(ctx context.Context, c *client.Client, r *Run, bucket string) {
	var roles []*client.ObjectStorageRole
	var accounts []*client.StorageAccount

	if err := r.op(oc, "object_storage.roles.list", func() error {
		var lerr error
		roles, lerr = c.ObjectStorage().Role().List(ctx)
		return lerr
	}); err != nil {
		r.skip(oc, "object_storage.acl.grant")
		r.skip(oc, "object_storage.acl.read")
		r.skip(oc, "object_storage.acl.revoke")
		return
	}
	if err := r.op(oc, "object_storage.storage_accounts.list", func() error {
		var lerr error
		accounts, lerr = c.ObjectStorage().StorageAccount().List(ctx)
		return lerr
	}); err != nil {
		r.skip(oc, "object_storage.acl.grant")
		r.skip(oc, "object_storage.acl.read")
		r.skip(oc, "object_storage.acl.revoke")
		return
	}

	if len(roles) == 0 || len(accounts) == 0 {
		r.skip(oc, "object_storage.acl.grant")
		r.skip(oc, "object_storage.acl.read")
		r.skip(oc, "object_storage.acl.revoke")
		return
	}

	role := roles[0].Name
	account := accounts[0].Name

	granted := false
	if err := r.op(oc, "object_storage.acl.grant", func() error {
		activityID, gerr := c.ObjectStorage().ACLEntry().Grant(ctx, bucket, role, account)
		if gerr != nil {
			return gerr
		}
		if activityID != "" {
			if _, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter); werr != nil {
				return werr
			}
		}
		granted = true
		return nil
	}); err != nil {
		r.skip(oc, "object_storage.acl.read")
		r.skip(oc, "object_storage.acl.revoke")
		return
	}

	if granted {
		r.Cleanup.Register(fmt.Sprintf("object_storage.acl revoke %s/%s/%s", bucket, role, account), func(tctx context.Context) error {
			activityID, rerr := c.ObjectStorage().ACLEntry().Revoke(tctx, bucket, role, account)
			if rerr != nil {
				return rerr
			}
			if activityID == "" {
				return nil
			}
			_, werr := c.Activity().WaitForCompletion(tctx, activityID, silentWaiter)
			return werr
		})
	}

	_ = r.op(oc, "object_storage.acl.read", func() error {
		_, rerr := c.ObjectStorage().ACL().ListByBucket(ctx, bucket)
		return rerr
	})

	_ = r.op(oc, "object_storage.acl.revoke", func() error {
		activityID, rerr := c.ObjectStorage().ACLEntry().Revoke(ctx, bucket, role, account)
		if rerr != nil {
			return rerr
		}
		if activityID == "" {
			return nil
		}
		_, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
		return werr
	})
}
