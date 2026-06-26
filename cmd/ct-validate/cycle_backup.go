package main

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// backupCycle is a deliberately READ-ONLY low-risk cycle: it lists SLA
// policies (and reads the first), plus the backup sites, storages and SPP
// servers inventory.
//
// It is Kind=Read on purpose. The brief allowed an assign/unassign SLA write
// step ONLY if proven safely reversible. It is NOT implemented here because:
//   - assigning an SLA policy with no sub-policies caused a hang on tenants
//     without backup (issue #306, fixed in the provider but still a sharp edge
//     against a shared API); and
//   - the client surface exposes no clean, idempotent unassign that this
//     harness can prove reversible offline.
//
// TODO(#316): add a gated SLA assign/unassign sub-cycle once a safely
// reversible assign/unassign pair is confirmed against a disposable tenant.
type backupCycle struct{}

func (backupCycle) Name() string { return "backup" }
func (backupCycle) Kind() Kind   { return KindRead }

func (bc backupCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	var policies []*client.BackupSLAPolicy
	_ = r.op(bc, "backup.sla_policies.list", func() error {
		var err error
		policies, err = c.Backup().SLAPolicy().List(ctx, nil)
		return err
	})
	if len(policies) > 0 {
		_ = r.op(bc, "backup.sla_policies.read", func() error {
			_, err := c.Backup().SLAPolicy().Read(ctx, policies[0].ID)
			return err
		})
	} else {
		r.skip(bc, "backup.sla_policies.read")
	}

	_ = r.op(bc, "backup.sites.list", func() error {
		_, err := c.Backup().Site().List(ctx)
		return err
	})
	_ = r.op(bc, "backup.storages.list", func() error {
		_, err := c.Backup().Storage().List(ctx)
		return err
	})
	_ = r.op(bc, "backup.spp_servers.list", func() error {
		_, err := c.Backup().SPPServer().List(ctx, nil)
		return err
	})
	return nil
}
