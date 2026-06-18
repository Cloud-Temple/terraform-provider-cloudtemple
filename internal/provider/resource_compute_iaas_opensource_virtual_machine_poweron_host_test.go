package provider

import (
	"errors"
	"testing"
)

// TestResolveOpenIaaSPowerOnHostID pins the #356 power-on host resolution and
// proves freshness via an injected readLiveHost (call-counted): a configured
// host_id never reads live, an unconfigured host_id reads live exactly once and
// returns the FRESH host (never the stale state value), an empty live host is
// returned as "" (HostId omitted), and a live-read error is surfaced without any
// fallback to the stale value.
func TestResolveOpenIaaSPowerOnHostID(t *testing.T) {
	t.Run("configured host_id returns the user's host without reading live", func(t *testing.T) {
		reads := 0
		got, err := resolveOpenIaaSPowerOnHostID(true, "host-b", func() (string, error) {
			reads++
			return "live-x", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "host-b" {
			t.Fatalf("got %q, want host-b (the configured host)", got)
		}
		if reads != 0 {
			t.Fatalf("a configured host_id must NOT read live, got %d reads", reads)
		}
	})

	t.Run("unconfigured host_id reads live once and returns the fresh host", func(t *testing.T) {
		reads := 0
		got, err := resolveOpenIaaSPowerOnHostID(false, "stale-state-host", func() (string, error) {
			reads++
			return "live-x", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "live-x" {
			t.Fatalf("got %q, want live-x (the live host, never the stale state value)", got)
		}
		if reads != 1 {
			t.Fatalf("an unconfigured host_id must read live exactly once, got %d", reads)
		}
	})

	t.Run("unconfigured with empty live host returns empty so HostId is omitted", func(t *testing.T) {
		got, err := resolveOpenIaaSPowerOnHostID(false, "stale-state-host", func() (string, error) {
			return "", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Fatalf("got %q, want \"\" so HostId is omitted (platform places the VM)", got)
		}
	})

	t.Run("unconfigured surfaces the live-read error and never falls back to the stale value", func(t *testing.T) {
		got, err := resolveOpenIaaSPowerOnHostID(false, "stale-state-host", func() (string, error) {
			return "", errors.New("boom")
		})
		if err == nil {
			t.Fatal("a live-read error must be surfaced")
		}
		if got != "" {
			t.Fatalf("on a live-read error the resolver must NOT fall back to the stale state value, got %q", got)
		}
	})
}
