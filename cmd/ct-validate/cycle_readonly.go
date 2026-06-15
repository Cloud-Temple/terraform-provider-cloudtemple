package main

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// readonlyCycle exercises the primary List (and a Read-by-id on the first
// item, where one exists) of every one of the eight services. It NEVER
// mutates and NEVER registers cleanup. It is the always-on safe cycle and the
// default selection.
//
// Each endpoint is recorded under a stable name (service.resource.list /
// .read) so the report can pinpoint exactly which API surface squeaks.
type readonlyCycle struct{}

func (readonlyCycle) Name() string { return "readonly" }
func (readonlyCycle) Kind() Kind   { return KindRead }

func (rc readonlyCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	// Tenant/company context is needed by a few IAM filters. A failure to read
	// it is recorded but does not abort the rest of the read sweep.
	var tenantID, companyID string
	_ = r.op(rc, "iam.token.read", func() error {
		tok, err := c.Token(ctx)
		if err != nil {
			return err
		}
		tenantID = tok.TenantID()
		companyID = tok.CompanyID()
		return nil
	})

	rc.runIAM(ctx, c, r, tenantID, companyID)
	rc.runVPC(ctx, c, r)
	rc.runCompute(ctx, c, r)
	rc.runBackup(ctx, c, r)
	rc.runObjectStorage(ctx, c, r)
	rc.runMarketplace(ctx, c, r)
	rc.runTag(ctx, c, r)
	rc.runActivity(ctx, c, r)
	return nil
}

func (rc readonlyCycle) runActivity(ctx context.Context, c *client.Client, r *Run) {
	var activities []*client.Activity
	_ = r.op(rc, "activity.activities.list", func() error {
		var err error
		activities, err = c.Activity().List(ctx, nil)
		return err
	})
	if len(activities) > 0 {
		_ = r.op(rc, "activity.activities.read", func() error {
			_, err := c.Activity().Read(ctx, activities[0].ID)
			return err
		})
	} else {
		r.skip(rc, "activity.activities.read")
	}
}

func (rc readonlyCycle) runIAM(ctx context.Context, c *client.Client, r *Run, tenantID, companyID string) {
	var users []*client.User
	_ = r.op(rc, "iam.users.list", func() error {
		var err error
		users, err = c.IAM().User().List(ctx, &client.UserFilter{CompanyID: companyID})
		return err
	})
	if len(users) > 0 {
		_ = r.op(rc, "iam.users.read", func() error {
			_, err := c.IAM().User().Read(ctx, users[0].ID)
			return err
		})
	} else {
		r.skip(rc, "iam.users.read")
	}

	var roles []*client.Role
	_ = r.op(rc, "iam.roles.list", func() error {
		var err error
		roles, err = c.IAM().Role().List(ctx)
		return err
	})
	if len(roles) > 0 {
		_ = r.op(rc, "iam.roles.read", func() error {
			_, err := c.IAM().Role().Read(ctx, roles[0].ID)
			return err
		})
	} else {
		r.skip(rc, "iam.roles.read")
	}

	_ = r.op(rc, "iam.tenants.list", func() error {
		_, err := c.IAM().Tenant().List(ctx)
		return err
	})
	_ = r.op(rc, "iam.features.list", func() error {
		_, err := c.IAM().Feature().List(ctx)
		return err
	})
	if tenantID != "" {
		_ = r.op(rc, "iam.feature_assignments.list", func() error {
			_, err := c.IAM().Feature().ListAssignments(ctx, tenantID)
			return err
		})
	} else {
		r.skip(rc, "iam.feature_assignments.list")
	}
	if companyID != "" {
		_ = r.op(rc, "iam.companies.read", func() error {
			_, err := c.IAM().Company().Read(ctx, companyID)
			return err
		})
	} else {
		r.skip(rc, "iam.companies.read")
	}
	_ = r.op(rc, "iam.pat.list", func() error {
		_, err := c.IAM().PAT().List(ctx)
		return err
	})
}

func (rc readonlyCycle) runVPC(ctx context.Context, c *client.Client, r *Run) {
	var pns []*client.PrivateNetwork
	_ = r.op(rc, "vpc.vpc.list", func() error {
		_, err := c.VPC().VPC().List(ctx)
		return err
	})
	_ = r.op(rc, "vpc.private_networks.list", func() error {
		var err error
		pns, err = c.VPC().PrivateNetwork().List(ctx, nil)
		return err
	})
	if len(pns) > 0 {
		_ = r.op(rc, "vpc.private_networks.read", func() error {
			_, err := c.VPC().PrivateNetwork().Read(ctx, pns[0].ID)
			return err
		})
		// Static IPs need a parent private-network id: list the parent first,
		// then list its static IPs.
		_ = r.op(rc, "vpc.static_ips.list", func() error {
			_, err := c.VPC().StaticIP().List(ctx, pns[0].ID, nil)
			return err
		})
	} else {
		r.skip(rc, "vpc.private_networks.read")
		r.skip(rc, "vpc.static_ips.list")
	}
	var fips []*client.FloatingIP
	_ = r.op(rc, "vpc.floating_ips.list", func() error {
		var err error
		fips, err = c.VPC().FloatingIP().List(ctx, nil)
		return err
	})
	if len(fips) > 0 {
		_ = r.op(rc, "vpc.floating_ips.read", func() error {
			_, err := c.VPC().FloatingIP().Read(ctx, fips[0].ID)
			return err
		})
	} else {
		r.skip(rc, "vpc.floating_ips.read")
	}
}

func (rc readonlyCycle) runCompute(ctx context.Context, c *client.Client, r *Run) {
	// VMware compute. These lists need a machine_manager_id scope that this
	// read-only smoke test cannot discover (the client exposes no VMware
	// machine-manager listing). Probe the entry list once: a 4xx means VMware is
	// not available/usable on this tenant, so SKIP the rest of the block rather
	// than emit a false "squeak" per endpoint. A 5xx/timeout/transient still
	// surfaces as a real failure on the probe.
	var vms []*client.VirtualMachine
	vmwareErr := r.op(rc, "compute.virtual_machines.list", func() error {
		var err error
		vms, err = c.Compute().VirtualMachine().List(ctx, nil)
		return err
	})
	vmwareRest := []string{
		"compute.virtual_machines.read", "compute.datastores.list",
		"compute.hosts.list", "compute.networks.list",
		"compute.virtual_datacenters.list", "compute.folders.list",
		"compute.virtual_disks.list",
	}
	if categorize(vmwareErr) == CategoryHTTP4xx {
		for _, ep := range vmwareRest {
			r.skip(rc, ep)
		}
	} else {
		if len(vms) > 0 {
			_ = r.op(rc, "compute.virtual_machines.read", func() error {
				_, err := c.Compute().VirtualMachine().Read(ctx, vms[0].ID)
				return err
			})
		} else {
			r.skip(rc, "compute.virtual_machines.read")
		}
		_ = r.op(rc, "compute.datastores.list", func() error {
			_, err := c.Compute().Datastore().List(ctx, nil)
			return err
		})
		_ = r.op(rc, "compute.hosts.list", func() error {
			_, err := c.Compute().Host().List(ctx, nil)
			return err
		})
		_ = r.op(rc, "compute.networks.list", func() error {
			_, err := c.Compute().Network().List(ctx, nil)
			return err
		})
		_ = r.op(rc, "compute.virtual_datacenters.list", func() error {
			_, err := c.Compute().VirtualDatacenter().List(ctx, nil)
			return err
		})
		_ = r.op(rc, "compute.folders.list", func() error {
			_, err := c.Compute().Folder().List(ctx, nil)
			return err
		})
		_ = r.op(rc, "compute.virtual_disks.list", func() error {
			_, err := c.Compute().VirtualDisk().List(ctx, nil)
			return err
		})
	}

	// OpenIaaS compute. Discover the machine manager FIRST: EVERY OpenIaaS list
	// (virtual_machines, networks, storage_repositories, templates, hosts,
	// pools) is scoped by machine_manager_id — the API answers 5xx (sometimes
	// intermittently) without it. Scope them all to the first machine manager;
	// if the tenant has none, skip them.
	var mms []*client.OpenIaaSMachineManager
	_ = r.op(rc, "compute.openiaas.machine_managers.list", func() error {
		var err error
		mms, err = c.Compute().OpenIaaS().MachineManager().List(ctx)
		return err
	})
	if len(mms) == 0 {
		for _, ep := range []string{
			"compute.openiaas.virtual_machines.list", "compute.openiaas.virtual_machines.read",
			"compute.openiaas.networks.list", "compute.openiaas.storage_repositories.list",
			"compute.openiaas.templates.list", "compute.openiaas.hosts.list", "compute.openiaas.pools.list",
		} {
			r.skip(rc, ep)
		}
		return
	}
	// Smoke read-only scope: the FIRST machine manager. On a multi-machine-manager
	// tenant this maps one MM, not all of them (a fuller sweep would iterate mms).
	mmID := mms[0].ID
	var oVMs []*client.OpenIaaSVirtualMachine
	_ = r.op(rc, "compute.openiaas.virtual_machines.list", func() error {
		var err error
		oVMs, err = c.Compute().OpenIaaS().VirtualMachine().ListStrict(ctx, &client.OpenIaaSVirtualMachineFilter{MachineManagerID: mmID})
		return err
	})
	if len(oVMs) > 0 {
		_ = r.op(rc, "compute.openiaas.virtual_machines.read", func() error {
			_, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, oVMs[0].ID)
			return err
		})
	} else {
		r.skip(rc, "compute.openiaas.virtual_machines.read")
	}
	_ = r.op(rc, "compute.openiaas.networks.list", func() error {
		_, err := c.Compute().OpenIaaS().Network().List(ctx, &client.OpenIaaSNetworkFilter{MachineManagerID: mmID})
		return err
	})
	_ = r.op(rc, "compute.openiaas.storage_repositories.list", func() error {
		_, err := c.Compute().OpenIaaS().StorageRepository().List(ctx, &client.StorageRepositoryFilter{MachineManagerId: mmID})
		return err
	})
	_ = r.op(rc, "compute.openiaas.templates.list", func() error {
		_, err := c.Compute().OpenIaaS().Template().List(ctx, &client.OpenIaaSTemplateFilter{MachineManagerId: mmID})
		return err
	})
	_ = r.op(rc, "compute.openiaas.hosts.list", func() error {
		_, err := c.Compute().OpenIaaS().Host().List(ctx, &client.OpenIaasHostFilter{MachineManagerId: mmID})
		return err
	})
	_ = r.op(rc, "compute.openiaas.pools.list", func() error {
		_, err := c.Compute().OpenIaaS().Pool().List(ctx, &client.OpenIaasPoolFilter{MachineManagerId: mmID})
		return err
	})
}

func (rc readonlyCycle) runBackup(ctx context.Context, c *client.Client, r *Run) {
	// Backup may be absent on a tenant. Probe once; a 4xx means "not available
	// on this tenant", so skip the rest instead of emitting a squeak per
	// endpoint. A 5xx/timeout still surfaces as a real failure on the probe.
	backupErr := r.op(rc, "backup.sla_policies.list", func() error {
		_, err := c.Backup().SLAPolicy().List(ctx, nil)
		return err
	})
	if categorize(backupErr) == CategoryHTTP4xx {
		for _, ep := range []string{"backup.sites.list", "backup.storages.list", "backup.spp_servers.list"} {
			r.skip(rc, ep)
		}
		return
	}
	_ = r.op(rc, "backup.sites.list", func() error {
		_, err := c.Backup().Site().List(ctx)
		return err
	})
	_ = r.op(rc, "backup.storages.list", func() error {
		_, err := c.Backup().Storage().List(ctx)
		return err
	})
	_ = r.op(rc, "backup.spp_servers.list", func() error {
		_, err := c.Backup().SPPServer().List(ctx, nil)
		return err
	})
}

func (rc readonlyCycle) runObjectStorage(ctx context.Context, c *client.Client, r *Run) {
	var buckets []*client.Bucket
	_ = r.op(rc, "object_storage.buckets.list", func() error {
		var err error
		buckets, err = c.ObjectStorage().Bucket().List(ctx)
		return err
	})
	if len(buckets) > 0 {
		_ = r.op(rc, "object_storage.buckets.read", func() error {
			_, err := c.ObjectStorage().Bucket().Read(ctx, buckets[0].Name)
			return err
		})
	} else {
		r.skip(rc, "object_storage.buckets.read")
	}
	_ = r.op(rc, "object_storage.storage_accounts.list", func() error {
		_, err := c.ObjectStorage().StorageAccount().List(ctx)
		return err
	})
	_ = r.op(rc, "object_storage.namespace.read", func() error {
		_, err := c.ObjectStorage().Namespace().Read(ctx)
		return err
	})
}

func (rc readonlyCycle) runMarketplace(ctx context.Context, c *client.Client, r *Run) {
	var items []*client.MarketplaceItem
	_ = r.op(rc, "marketplace.items.list", func() error {
		var err error
		items, err = c.Marketplace().Item().List(ctx)
		return err
	})
	if len(items) > 0 {
		_ = r.op(rc, "marketplace.items.read", func() error {
			_, err := c.Marketplace().Item().Read(ctx, items[0].ID)
			return err
		})
	} else {
		r.skip(rc, "marketplace.items.read")
	}
}

func (rc readonlyCycle) runTag(ctx context.Context, c *client.Client, r *Run) {
	// The Tag service exposes only per-resource Read/Create/Delete (no global
	// list endpoint). A safe read needs a resource id; we reuse the first
	// floating IP id if one exists, otherwise skip. This keeps the read sweep
	// honest rather than inventing a fake id.
	var fips []*client.FloatingIP
	_ = r.op(rc, "tag.floating_ips.list_for_resource", func() error {
		var err error
		fips, err = c.VPC().FloatingIP().List(ctx, nil)
		return err
	})
	if len(fips) > 0 {
		_ = r.op(rc, "tag.resource.read", func() error {
			_, err := c.Tag().Resource().Read(ctx, fips[0].ID)
			return err
		})
	} else {
		r.skip(rc, "tag.resource.read")
	}
}
