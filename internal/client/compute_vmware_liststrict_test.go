package client

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

// captureHandler records the inbound request and replies with status/body.
func captureHandler(status int, body string, gotMethod, gotPath *string, gotQuery *url.Values) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*gotMethod = r.Method
		*gotPath = r.URL.Path
		*gotQuery = r.URL.Query()
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// runListStrictRejections asserts that every non-200 / unusable answer is an
// error: a strict listing must fail closed so it can never wrongly drive a
// state deletion (#281).
func runListStrictRejections(t *testing.T, call func(c *Client) error) {
	t.Helper()
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"201 Created", http.StatusCreated, "[]"},
		{"206 Partial Content", http.StatusPartialContent, "[]"},
		{"403 Forbidden", http.StatusForbidden, "[]"},
		{"500 Internal Server Error", http.StatusInternalServerError, "[]"},
		{"malformed 200", http.StatusOK, "{not json"},
		{"empty 200 body", http.StatusOK, ""},
	} {
		tc := tc
		t.Run(tc.name+" is rejected", func(t *testing.T) {
			c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			if err := call(c); err == nil {
				t.Fatalf("status %d / body %q must be rejected as unusable evidence", tc.status, tc.body)
			}
		})
	}
}

func TestVirtualMachineListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 scopes by machineManagerId and returns the parsed VMs", func(t *testing.T) {
		var method, path string
		var query url.Values
		c := newPATTestClient(t, captureHandler(http.StatusOK, `[{"id":"vm-1"},{"id":"vm-2"}]`, &method, &path, &query))
		vms, err := c.Compute().VirtualMachine().ListStrict(ctx, &VirtualMachineFilter{MachineManagerID: "mm-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vms) != 2 || vms[0].ID != "vm-1" {
			t.Fatalf("unexpected vms: %+v", vms)
		}
		if method != http.MethodGet || path != "/compute/v1/vcenters/virtual_machines" {
			t.Fatalf("unexpected request: %s %s", method, path)
		}
		if query.Get("machineManagerId") != "mm-1" {
			t.Fatalf("expected machineManagerId=mm-1, got query %v", query)
		}
	})

	t.Run("an empty array is a valid empty listing", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		})
		vms, err := c.Compute().VirtualMachine().ListStrict(ctx, &VirtualMachineFilter{MachineManagerID: "mm-1"})
		if err != nil || len(vms) != 0 {
			t.Fatalf("an empty array must be accepted, got vms=%v err=%v", vms, err)
		}
	})

	runListStrictRejections(t, func(c *Client) error {
		_, err := c.Compute().VirtualMachine().ListStrict(ctx, &VirtualMachineFilter{MachineManagerID: "mm-1"})
		return err
	})
}

func TestVirtualDiskListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 scopes by virtualMachineId and returns the parsed disks", func(t *testing.T) {
		var method, path string
		var query url.Values
		c := newPATTestClient(t, captureHandler(http.StatusOK, `[{"id":"disk-1"}]`, &method, &path, &query))
		disks, err := c.Compute().VirtualDisk().ListStrict(ctx, &VirtualDiskFilter{VirtualMachineID: "vm-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(disks) != 1 || disks[0].ID != "disk-1" {
			t.Fatalf("unexpected disks: %+v", disks)
		}
		if method != http.MethodGet || path != "/compute/v1/vcenters/virtual_disks" {
			t.Fatalf("unexpected request: %s %s", method, path)
		}
		if query.Get("virtualMachineId") != "vm-1" {
			t.Fatalf("expected virtualMachineId=vm-1, got query %v", query)
		}
	})

	runListStrictRejections(t, func(c *Client) error {
		_, err := c.Compute().VirtualDisk().ListStrict(ctx, &VirtualDiskFilter{VirtualMachineID: "vm-1"})
		return err
	})
}

func TestNetworkAdapterListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 scopes by virtualMachineId and returns the parsed adapters", func(t *testing.T) {
		var method, path string
		var query url.Values
		c := newPATTestClient(t, captureHandler(http.StatusOK, `[{"id":"nic-1"}]`, &method, &path, &query))
		adapters, err := c.Compute().NetworkAdapter().ListStrict(ctx, &NetworkAdapterFilter{VirtualMachineID: "vm-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(adapters) != 1 || adapters[0].ID != "nic-1" {
			t.Fatalf("unexpected adapters: %+v", adapters)
		}
		if method != http.MethodGet || path != "/compute/v1/vcenters/network_adapters" {
			t.Fatalf("unexpected request: %s %s", method, path)
		}
		if query.Get("virtualMachineId") != "vm-1" {
			t.Fatalf("expected virtualMachineId=vm-1, got query %v", query)
		}
	})

	runListStrictRejections(t, func(c *Client) error {
		_, err := c.Compute().NetworkAdapter().ListStrict(ctx, &NetworkAdapterFilter{VirtualMachineID: "vm-1"})
		return err
	})
}

func TestVirtualControllerListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 scopes by virtualMachineId only and never sends a type filter", func(t *testing.T) {
		var method, path string
		var query url.Values
		c := newPATTestClient(t, captureHandler(http.StatusOK, `[{"id":"ctrl-1"}]`, &method, &path, &query))
		controllers, err := c.Compute().VirtualController().ListStrict(ctx, &VirtualControllerFilter{VirtualMachineId: "vm-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(controllers) != 1 || controllers[0].ID != "ctrl-1" {
			t.Fatalf("unexpected controllers: %+v", controllers)
		}
		if method != http.MethodGet || path != "/compute/v1/vcenters/virtual_controllers" {
			t.Fatalf("unexpected request: %s %s", method, path)
		}
		if query.Get("virtualMachineId") != "vm-1" {
			t.Fatalf("expected virtualMachineId=vm-1, got query %v", query)
		}
		// The deletion-evidence path must list ALL controllers of the VM: a
		// stray type filter would narrow the listing and could hide a present
		// controller, turning a liveness check into a false absence.
		if query.Get("types") != "" || len(query["types[]"]) != 0 {
			t.Fatalf("the evidence listing must not send a type filter, got query %v", query)
		}
	})

	runListStrictRejections(t, func(c *Client) error {
		_, err := c.Compute().VirtualController().ListStrict(ctx, &VirtualControllerFilter{VirtualMachineId: "vm-1"})
		return err
	})
}
