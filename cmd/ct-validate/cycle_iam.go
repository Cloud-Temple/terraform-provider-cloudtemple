package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// iamPATCycle drives a personal access token lifecycle:
//
//	(pick a valid role) -> create PAT -> read it -> delete it
//
// The PAT teardown is registered the moment the create returns a usable token,
// so an abort still deletes it. A PAT is a credential: leaving one orphaned is
// a security issue, which is exactly why teardown registration is immediate.
type iamPATCycle struct{}

func (iamPATCycle) Name() string { return "iam_pat" }
func (iamPATCycle) Kind() Kind   { return KindWrite }

func (ic iamPATCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	role, ok := ic.pickRole(ctx, c, r)
	if !ok {
		r.skip(ic, "iam.pat.create")
		r.skip(ic, "iam.pat.read")
		r.skip(ic, "iam.pat.delete")
		return nil
	}

	name := fmt.Sprintf("ct-validate-%d-%d", r.Iteration, r.Worker)
	// Short-lived token: 1 hour from now, in epoch milliseconds (the API and
	// the provider use milliseconds — see resource_iam_personal_access_token).
	expiration := int(time.Now().Add(time.Hour).UTC().UnixMilli())

	// F3: register the PAT teardown BEFORE the create/decode. A PAT is a live
	// credential; a created-but-undecoded one (201 received, body decode fails) is
	// a security orphan. The teardown deletes by id once resolved, else falls back
	// to find-by-name-and-delete. The ref is shared so the cycle can fill in the
	// resolved id AFTER registration without re-registering.
	ref := &patTeardownRef{Name: name}
	registerPATTeardown(r.Cleanup, iamPATSeam{c}, ref)

	if err := r.op(ic, "iam.pat.create", func() error {
		tok, cerr := c.IAM().PAT().Create(ctx, name, []string{role}, expiration)
		if cerr != nil {
			return cerr
		}
		if tok == nil || tok.ID == "" {
			return fmt.Errorf("PAT created but no id returned")
		}
		ref.ID = tok.ID
		ref.Resolved = true
		return nil
	}); err != nil || !ref.Resolved {
		// Create failed or returned no usable id. The pre-registered teardown
		// still removes a created-but-undecoded PAT by find-by-name.
		return err
	}

	_ = r.op(ic, "iam.pat.read", func() error {
		_, rerr := c.IAM().PAT().Read(ctx, ref.ID)
		return rerr
	})

	_ = r.op(ic, "iam.pat.delete", func() error {
		return c.IAM().PAT().Delete(ctx, ref.ID)
	})
	return nil
}

// pickRole returns a role id usable for a new PAT. It first reuses the roles of
// an existing PAT (guaranteed assignable to this principal), falling back to
// the tenant's IAM roles. It returns ("", false) when no role can be found, so
// the cycle skips rather than guessing.
func (ic iamPATCycle) pickRole(ctx context.Context, c *client.Client, r *Run) (string, bool) {
	var pats []*client.Token
	if err := r.op(ic, "iam.pat.list", func() error {
		var lerr error
		pats, lerr = c.IAM().PAT().List(ctx)
		return lerr
	}); err == nil {
		for _, p := range pats {
			if p != nil && len(p.Roles) > 0 && p.Roles[0] != "" {
				return p.Roles[0], true
			}
		}
	}

	var roles []*client.Role
	if err := r.op(ic, "iam.roles.list", func() error {
		var lerr error
		roles, lerr = c.IAM().Role().List(ctx)
		return lerr
	}); err == nil {
		for _, role := range roles {
			if role != nil && role.ID != "" {
				return role.ID, true
			}
		}
	}
	return "", false
}
