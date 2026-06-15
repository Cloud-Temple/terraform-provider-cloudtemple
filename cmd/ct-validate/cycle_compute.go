package main

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// computeOpenIaaSCycle validates the OpenIaaS compute READ surface that a full
// VM lifecycle would depend on: machine managers, storage repositories,
// networks, templates, and the marketplace items used as deploy sources
// (including a Read of the first item to surface its adapter count). It also
// reads back the first existing VM.
//
// It is intentionally READ-ONLY (Kind=Read) in this pass.
//
// TODO(#316): implement the full, GATED heavy VM lifecycle
// (deploy from a marketplace item -> power on -> add/remove a network adapter
// -> power off -> delete). The full lifecycle is deferred here because shipping
// it now would risk leaving an orphan VM, which violates the never-orphan
// doctrine. Three known prerequisites must be wired and asserted first:
//  1. the number of inline os_network_adapter blocks MUST equal the marketplace
//     item's adapter count;
//  2. if cloud_init is present, cloudConfig is REQUIRED;
//  3. storage_repository_id is REQUIRED together with marketplace_item_id.
//
// The client surface needed for the lifecycle DOES exist
// (Marketplace().Item().DeployOpenIaasItem, OpenIaaS().VirtualMachine().Power /
// .Delete, OpenIaaS().NetworkAdapter().Create / .Delete), so #316 is a matter
// of wiring substrate selection + the three prerequisites + bounded waits +
// teardown registration, not of missing client methods.
type computeOpenIaaSCycle struct{}

func (computeOpenIaaSCycle) Name() string { return "compute_openiaas" }
func (computeOpenIaaSCycle) Kind() Kind   { return KindRead }

func (cc computeOpenIaaSCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	_ = r.op(cc, "compute.openiaas.machine_managers.list", func() error {
		_, err := c.Compute().OpenIaaS().MachineManager().List(ctx)
		return err
	})
	_ = r.op(cc, "compute.openiaas.storage_repositories.list", func() error {
		_, err := c.Compute().OpenIaaS().StorageRepository().List(ctx, nil)
		return err
	})
	_ = r.op(cc, "compute.openiaas.networks.list", func() error {
		_, err := c.Compute().OpenIaaS().Network().List(ctx, nil)
		return err
	})
	_ = r.op(cc, "compute.openiaas.templates.list", func() error {
		_, err := c.Compute().OpenIaaS().Template().List(ctx, nil)
		return err
	})

	var items []*client.MarketplaceItem
	_ = r.op(cc, "marketplace.items.list", func() error {
		var err error
		items, err = c.Marketplace().Item().List(ctx)
		return err
	})
	if len(items) > 0 {
		_ = r.op(cc, "marketplace.items.read", func() error {
			_, err := c.Marketplace().Item().Read(ctx, items[0].ID)
			return err
		})
	} else {
		r.skip(cc, "marketplace.items.read")
	}

	var vms []*client.OpenIaaSVirtualMachine
	_ = r.op(cc, "compute.openiaas.virtual_machines.list", func() error {
		var err error
		vms, err = c.Compute().OpenIaaS().VirtualMachine().ListStrict(ctx, nil)
		return err
	})
	if len(vms) > 0 {
		_ = r.op(cc, "compute.openiaas.virtual_machines.read", func() error {
			_, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, vms[0].ID)
			return err
		})
	} else {
		r.skip(cc, "compute.openiaas.virtual_machines.read")
	}
	return nil
}
