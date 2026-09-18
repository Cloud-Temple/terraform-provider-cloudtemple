package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// The defect these tests pin was measured live on DEV on 2026-08-21, during a
// 25-way concurrent OpenIaaS create: a VM requested 10.0.5.103, an address already
// registered to another tenant resource. The platform did NOT refuse. It created
// the VM, reported success, and did not register the address — while Terraform
// recorded 10.0.5.103 in the state. Since ip_address is write-only, nothing ever
// detects that afterwards. Only a pre-flight can catch it.
func TestRejectInlineAdapterIPAlreadyRegistered(t *testing.T) {
	ctx := context.Background()
	const ownerVM = "vm-self"

	holder := func(ip, vmID, source, mac string, floating bool) *client.StaticIP {
		s := &client.StaticIP{IPAddress: ip, Source: source, MacAddress: mac}
		if vmID != "" {
			s.VirtualMachine = &client.BaseObject{ID: vmID}
		}
		if floating {
			s.FloatingIP = &client.StaticIPFloatingIP{}
		}
		return s
	}
	// conflicts maps "networkID|ip" to the registration holding it.
	checker := func(conflicts map[string]*client.StaticIP, failOn string) inlineIPConflictFunc {
		return func(_ context.Context, networkID, ip string) (*client.StaticIP, error) {
			if failOn != "" && networkID == failOn {
				return nil, errors.New("vpc plane unreachable")
			}
			return conflicts[networkID+"|"+ip], nil
		}
	}

	cases := []struct {
		name        string
		configured  map[int]string
		networkAt   map[int]string
		conflicts   map[string]*client.StaticIP
		failOn      string
		owner       string
		ownerMACs   []string
		wantRefused bool
		wantIn      []string
		wantNotIn   []string
	}{
		{
			name:       "a free address passes",
			configured: map[int]string{0: "10.0.5.50"},
			networkAt:  map[int]string{0: "net-a"},
			conflicts:  map[string]*client.StaticIP{},
		},
		{
			name:       "an address held by ANOTHER resource is refused",
			configured: map[int]string{0: "10.0.5.103"},
			networkAt:  map[int]string{0: "net-a"},
			conflicts: map[string]*client.StaticIP{
				"net-a|10.0.5.103": holder("10.0.5.103", "someone-else", "vmi", "3a:ad:76:5d:e4:e9", true),
			},
			wantRefused: true,
			// The diagnostic must carry the index, the address, who holds it, and the
			// reason the platform will not tell you — that last part is what stops a
			// user from "fixing" it by retrying.
			wantIn: []string{"os_network_adapter[0]", "10.0.5.103", "ALREADY registered", "vmi", "someone-else", "3a:ad:76:5d:e4:e9", "floating IP", "silently", "write-only"},
		},
		{
			name:       "an address already held by THIS resource passes (steady-state update)",
			configured: map[int]string{0: "10.0.5.103"},
			networkAt:  map[int]string{0: "net-a"},
			conflicts: map[string]*client.StaticIP{
				"net-a|10.0.5.103": holder("10.0.5.103", ownerVM, "xoa", "aa:bb:cc:dd:ee:ff", false),
			},
			owner: ownerVM,
		},
		{
			// The Compute surfaces register by MAC and the platform does not always
			// link the machine (client fixtures: `virtualMachine: null` on xoa/vmware
			// rows). The VM's own adapter MAC is positive proof of ownership.
			name:       "an address held by one of THIS resource's own MACs, with NO machine link, passes (Compute update)",
			configured: map[int]string{0: "10.0.5.103"},
			networkAt:  map[int]string{0: "net-a"},
			conflicts: map[string]*client.StaticIP{
				"net-a|10.0.5.103": holder("10.0.5.103", "", "xoa", "AA:BB:CC:DD:EE:FF", false),
			},
			owner:     ownerVM,
			ownerMACs: []string{"11:22:33:44:55:66", "aa:bb:cc:dd:ee:ff"}, // case-insensitive
		},
		{
			name:       "an address held by a FOREIGN MAC, with no machine link, is refused on an update",
			configured: map[int]string{0: "10.0.5.103"},
			networkAt:  map[int]string{0: "net-a"},
			conflicts: map[string]*client.StaticIP{
				"net-a|10.0.5.103": holder("10.0.5.103", "", "xoa", "de:ad:be:ef:00:01", false),
			},
			owner:       ownerVM,
			ownerMACs:   []string{"aa:bb:cc:dd:ee:ff"},
			wantRefused: true,
			wantIn:      []string{"ALREADY registered", "de:ad:be:ef:00:01"},
		},
		{
			name:       "a registration with NEITHER machine link NOR MAC is not proven ours: refused",
			configured: map[int]string{0: "10.0.5.103"},
			networkAt:  map[int]string{0: "net-a"},
			conflicts: map[string]*client.StaticIP{
				"net-a|10.0.5.103": holder("10.0.5.103", "", "xoa", "", false),
			},
			owner:       ownerVM,
			ownerMACs:   []string{"aa:bb:cc:dd:ee:ff", ""},
			wantRefused: true,
		},
		{
			name:       "on a CREATE (no owner) even a same-address registration is a conflict, with the replacement hint",
			configured: map[int]string{0: "10.0.5.103"},
			networkAt:  map[int]string{0: "net-a"},
			conflicts: map[string]*client.StaticIP{
				"net-a|10.0.5.103": holder("10.0.5.103", ownerVM, "xoa", "", false),
			},
			owner:       "",
			wantRefused: true,
			// A create is also how a replacement recreates the VM, and the address may
			// then be held by the machine being replaced: the refusal must say what to
			// do in that case (#533), for both lifecycle orderings.
			wantIn: []string{"ALREADY registered", "destroy-before-create", "retry once the registration has disappeared", "create_before_destroy"},
		},
		{
			name:       "on an UPDATE (owner known) a foreign holder is refused WITHOUT the replacement hint",
			configured: map[int]string{0: "10.0.5.103"},
			networkAt:  map[int]string{0: "net-a"},
			conflicts: map[string]*client.StaticIP{
				"net-a|10.0.5.103": holder("10.0.5.103", "someone-else", "xoa", "", false),
			},
			owner:       ownerVM,
			wantRefused: true,
			wantIn:      []string{"ALREADY registered", "someone-else"},
			wantNotIn:   []string{"create_before_destroy"},
		},
		{
			name:        "a read failure REFUSES (fail closed): unreadable must never mean free",
			configured:  map[int]string{0: "10.0.5.50"},
			networkAt:   map[int]string{0: "net-a"},
			failOn:      "net-a",
			wantRefused: true,
			wantIn:      []string{"failed to verify", "10.0.5.50", "Refusing"},
		},
		{
			name:       "a block with no resolved network is left to the VPC precondition",
			configured: map[int]string{0: "10.0.5.50"},
			networkAt:  map[int]string{},
			conflicts:  map[string]*client.StaticIP{},
		},
		{
			name:       "the LOWEST conflicting index is reported, deterministically",
			configured: map[int]string{2: "10.0.5.72", 0: "10.0.5.70", 1: "10.0.5.71"},
			networkAt:  map[int]string{0: "net-a", 1: "net-b", 2: "net-c"},
			conflicts: map[string]*client.StaticIP{
				"net-b|10.0.5.71": holder("10.0.5.71", "other", "xoa", "", false),
				"net-c|10.0.5.72": holder("10.0.5.72", "other", "xoa", "", false),
			},
			wantRefused: true,
			wantIn:      []string{"os_network_adapter[1]", "10.0.5.71"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			networkIDAt := func(i int) string { return tc.networkAt[i] }
			diags := rejectInlineAdapterIPAlreadyRegistered(ctx, tc.configured, networkIDAt,
				checker(tc.conflicts, tc.failOn), tc.owner, tc.ownerMACs)
			if tc.wantRefused && !diags.HasError() {
				t.Fatalf("expected a refusal, got none — a request the platform would silently decline would go through")
			}
			if !tc.wantRefused && diags.HasError() {
				t.Fatalf("expected no refusal, got %q", diags[0].Summary)
			}
			if !tc.wantRefused {
				return
			}
			got := diags[0].Summary
			for _, want := range tc.wantIn {
				if !strings.Contains(got, want) {
					t.Fatalf("diagnostic is missing %q, which the user needs to act on it.\ngot: %s", want, got)
				}
			}
			for _, unwanted := range tc.wantNotIn {
				if strings.Contains(got, unwanted) {
					t.Fatalf("diagnostic must not contain %q here.\ngot: %s", unwanted, got)
				}
			}
		})
	}

	t.Run("a nil checker is a no-op, not a panic", func(t *testing.T) {
		if diags := rejectInlineAdapterIPAlreadyRegistered(ctx, map[int]string{0: "10.0.5.1"},
			func(int) string { return "net-a" }, nil, "", nil); diags != nil {
			t.Fatalf("expected no diagnostics with a nil checker, got %v", diags)
		}
	})
}

// TestStaticIPConflictChecker covers the composition layer: resolving the private
// network, memoising the listing, and refusing to treat an error as "free".
func TestStaticIPConflictChecker(t *testing.T) {
	ctx := context.Background()

	t.Run("finds the holder and memoises the listing per private network", func(t *testing.T) {
		listings := 0
		c := staticIPConflictChecker(
			func(_ context.Context, networkID string) (string, error) { return "pn-1", nil },
			func(_ context.Context, pnID string) ([]*client.StaticIP, error) {
				listings++
				return []*client.StaticIP{{IPAddress: "10.0.5.9", MacAddress: "x"}}, nil
			},
		)
		if h, err := c(ctx, "net-a", "10.0.5.9"); err != nil || h == nil {
			t.Fatalf("expected the holder, got h=%v err=%v", h, err)
		}
		if h, err := c(ctx, "net-b", "10.0.5.8"); err != nil || h != nil {
			t.Fatalf("expected no holder for a free address, got h=%v err=%v", h, err)
		}
		// Both blocks resolved to pn-1: re-reading it would multiply calls against an
		// API that has already been observed dropping requests under load.
		if listings != 1 {
			t.Fatalf("expected 1 listing for the same private network, got %d", listings)
		}
	})

	t.Run("a non-VPC network yields no conflict and no listing", func(t *testing.T) {
		listings := 0
		c := staticIPConflictChecker(
			func(_ context.Context, _ string) (string, error) { return "", nil },
			func(_ context.Context, _ string) ([]*client.StaticIP, error) { listings++; return nil, nil },
		)
		h, err := c(ctx, "net-plain", "10.0.5.9")
		if err != nil || h != nil {
			t.Fatalf("expected (nil, nil), got h=%v err=%v", h, err)
		}
		if listings != 0 {
			t.Fatalf("no IPAM listing should happen for a non-VPC network, got %d", listings)
		}
	})

	t.Run("errors propagate from both stages", func(t *testing.T) {
		resolveErr := staticIPConflictChecker(
			func(_ context.Context, _ string) (string, error) { return "", errors.New("network read failed") },
			func(_ context.Context, _ string) ([]*client.StaticIP, error) { return nil, nil },
		)
		if _, err := resolveErr(ctx, "net-a", "10.0.5.9"); err == nil {
			t.Fatal("a network-read failure must surface, not read as a free address")
		}
		listErr := staticIPConflictChecker(
			func(_ context.Context, _ string) (string, error) { return "pn-1", nil },
			func(_ context.Context, _ string) ([]*client.StaticIP, error) {
				return nil, errors.New("listing failed")
			},
		)
		if _, err := listErr(ctx, "net-a", "10.0.5.9"); err == nil {
			t.Fatal("a listing failure must surface, not read as a free address")
		}
	})
}

// TestVMInstanceClientFuncsWiresStaticIPLister guards the one fail-open seam in
// this feature: createVMInstanceWith skips the collision check when
// funcs.listStaticIPs is nil, so that older unit tests can build a partial struct.
// If the production wiring ever loses it, the check would silently stop running
// and the defect would return with no test failing. This is that test.
func TestVMInstanceClientFuncsWiresStaticIPLister(t *testing.T) {
	funcs := vmInstanceClientFuncs(&client.Client{})
	if funcs.listStaticIPs == nil {
		t.Fatal("vmInstanceClientFuncs does not wire listStaticIPs: the public_cloud_vm_instance create would silently skip the IPAM collision check, and an address the platform declines to register would be recorded in the state with no error")
	}
	if funcs.networkRead == nil {
		t.Fatal("vmInstanceClientFuncs does not wire networkRead, which the same preflight depends on")
	}
}

// TestInlineIPConflictOrNil pins the seam that made the plan-time hook panic the
// first time it was wired: unit tests drive Resource.Diff with a nil meta, and
// getClient() type-asserts. The hook must degrade to a no-op there — it is advisory,
// and the create/update precondition is the authoritative gate.
func TestInlineIPConflictOrNil(t *testing.T) {
	built := 0
	build := func(*client.Client) inlineIPConflictFunc {
		built++
		return func(context.Context, string, string) (*client.StaticIP, error) { return nil, nil }
	}

	if got := inlineIPConflictOrNil(nil, build); got != nil {
		t.Fatal("a nil meta must yield a nil checker, not a constructed one")
	}
	if got := inlineIPConflictOrNil("not-a-client", build); got != nil {
		t.Fatal("a meta that is not a *client.Client must yield a nil checker")
	}
	if got := inlineIPConflictOrNil((*client.Client)(nil), build); got != nil {
		t.Fatal("a typed-nil *client.Client must yield a nil checker")
	}
	if built != 0 {
		t.Fatalf("the builder must not run without a client, ran %d times", built)
	}
	if got := inlineIPConflictOrNil(&client.Client{}, build); got == nil {
		t.Fatal("a real client must yield a working checker")
	}
	if built != 1 {
		t.Fatalf("expected the builder to run once with a real client, got %d", built)
	}
}
