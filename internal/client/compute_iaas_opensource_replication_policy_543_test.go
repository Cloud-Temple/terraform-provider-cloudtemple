package client

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

// Issue #543 — `lastRun.start` / `lastRun.end` are Unix timestamps in
// MILLISECONDS. Declaring them `int` made every read that decodes
// OpenIaaSReplicationPolicy fail on the 386 and arm artifacts goreleaser
// publishes, where `int` is 32 bits:
//
//	json: cannot unmarshal number 1759830000000 into Go struct field … of type int
//
// The failure is not reproducible by executing this suite on a 64-bit host, so
// the gate here is the DECLARED width of the fields: reflect reports Int on the
// defect and Int64 on the fix, on every platform. The decode assertions pin the
// JSON shape (camelCase `lastRun`, nested `start`/`end`/`status`) that the width
// alone would not catch.
const (
	lastRunStartMs int64 = 1759830000000
	lastRunEndMs   int64 = 1759830012345
)

// assertLastRunIsSixtyFourBit is the portable RED gate: it fails on `int`
// (Kind Int) whatever the host word size, and passes only on int64.
func assertLastRunIsSixtyFourBit(t *testing.T, p *OpenIaaSReplicationPolicy) {
	t.Helper()
	for name, v := range map[string]any{"start": p.LastRun.Start, "end": p.LastRun.End} {
		if k := reflect.TypeOf(v).Kind(); k != reflect.Int64 {
			t.Fatalf("lastRun.%s is declared %s: a millisecond timestamp MUST be int64, "+
				"or the 386/arm builds refuse to decode it (#543)", name, k)
		}
	}
}

// TestReplicationPolicyLastRunDecodesMillisecondTimestamps covers the
// replication-policy routes: the single read, and the strict and lax listings.
func TestReplicationPolicyLastRunDecodesMillisecondTimestamps(t *testing.T) {
	ctx := context.Background()
	const body = `{"id":"pol-1","name":"nightly","lastRun":{"start":1759830000000,"end":1759830012345,"status":"SUCCESS"},"interval":{"hours":4,"minutes":30}}`

	t.Run("Read", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/compute/v1/open_iaas/replication/configurations/pol-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		})

		got, err := c.Compute().OpenIaaS().Replication().Policy().Read(ctx, "pol-1")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if got == nil {
			t.Fatal("Read returned no policy")
		}
		assertLastRunIsSixtyFourBit(t, got)
		if got.LastRun.Start != lastRunStartMs || got.LastRun.End != lastRunEndMs {
			t.Fatalf("lastRun decoded as {%d,%d}, want {%d,%d}",
				got.LastRun.Start, got.LastRun.End, lastRunStartMs, lastRunEndMs)
		}
		if got.LastRun.Status != "SUCCESS" {
			t.Fatalf("lastRun.status = %q, want SUCCESS", got.LastRun.Status)
		}
	})

	for _, tc := range []struct {
		name string
		call func(*Client) ([]*OpenIaaSReplicationPolicy, error)
	}{
		{"List", func(c *Client) ([]*OpenIaaSReplicationPolicy, error) {
			return c.Compute().OpenIaaS().Replication().Policy().List(ctx)
		}},
		{"ListStrict", func(c *Client) ([]*OpenIaaSReplicationPolicy, error) {
			return c.Compute().OpenIaaS().Replication().Policy().ListStrict(ctx)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/compute/v1/open_iaas/replication/configurations" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("[" + body + "]"))
			})

			got, err := tc.call(c)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if len(got) != 1 {
				t.Fatalf("%s returned %d policies, want 1", tc.name, len(got))
			}
			assertLastRunIsSixtyFourBit(t, got[0])
			if got[0].LastRun.Start != lastRunStartMs || got[0].LastRun.End != lastRunEndMs {
				t.Fatalf("lastRun decoded as {%d,%d}, want {%d,%d}",
					got[0].LastRun.Start, got[0].LastRun.End, lastRunStartMs, lastRunEndMs)
			}
		})
	}
}

// TestReplicationPolicyVirtualMachineReadDecodesMillisecondTimestamps covers the
// route that makes this a VM-wide outage rather than a replication-only one:
// resource_compute_iaas_opensource_virtual_machine.go resolves a VM's
// replication policy through /replication/virtual_machines/{id}/configurations
// on EVERY read and turns any error into a failed Read, so a decode refusal
// there fails `plan` for every Open IaaS virtual machine on a 32-bit build.
func TestReplicationPolicyVirtualMachineReadDecodesMillisecondTimestamps(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/compute/v1/open_iaas/replication/virtual_machines/vm-1/configurations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"pol-1","name":"nightly","lastRun":{"start":1759830000000,"end":1759830012345,"status":"SUCCESS"}}`))
	})

	got, err := c.Compute().OpenIaaS().Replication().Policy().VirtualMachine().Read(ctx, "vm-1")
	if err != nil {
		t.Fatalf("VirtualMachine().Read: %v", err)
	}
	if got == nil {
		t.Fatal("VirtualMachine().Read returned no policy")
	}
	assertLastRunIsSixtyFourBit(t, got)
	if got.LastRun.Start != lastRunStartMs || got.LastRun.End != lastRunEndMs {
		t.Fatalf("lastRun decoded as {%d,%d}, want {%d,%d}",
			got.LastRun.Start, got.LastRun.End, lastRunStartMs, lastRunEndMs)
	}
}
