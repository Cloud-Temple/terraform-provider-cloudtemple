package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

var errTransientVIF = errors.New("transient platform failure (test)")

// scriptedVIFFuncs builds an injectable API surface that counts the calls
// and never sleeps. waitErrs replay in order (nil = success), the last one
// repeating.
type vifCallCounts struct {
	reads, updates, waits, sleeps int
}

func scriptedVIFFuncs(counts *vifCallCounts, adapters []*client.OpenIaaSNetworkAdapter, waitErrs []error) vifUpdateFuncs {
	return vifUpdateFuncs{
		read: func(ctx context.Context) (*client.OpenIaaSNetworkAdapter, error) {
			i := counts.reads
			if i >= len(adapters) {
				i = len(adapters) - 1
			}
			counts.reads++
			return adapters[i], nil
		},
		update: func(ctx context.Context, req *client.UpdateOpenIaasNetworkAdapterRequest) (string, error) {
			counts.updates++
			return "act-1", nil
		},
		wait: func(ctx context.Context, activityID string) (*client.Activity, error) {
			i := counts.waits
			if i >= len(waitErrs) {
				i = len(waitErrs) - 1
			}
			counts.waits++
			return nil, waitErrs[i]
		},
		sleep: func(ctx context.Context, attempt int) error {
			counts.sleeps++
			return nil
		},
		isTransient: func(err error) bool { return errors.Is(err, errTransientVIF) },
	}
}

func divergedAdapter() *client.OpenIaaSNetworkAdapter {
	return &client.OpenIaaSNetworkAdapter{
		Network:        client.BaseObject{ID: "net-old"},
		MacAddress:     "aa:bb:cc:dd:ee:ff",
		TxChecksumming: true,
	}
}

func convergedAdapter() *client.OpenIaaSNetworkAdapter {
	return &client.OpenIaaSNetworkAdapter{
		Network:        client.BaseObject{ID: "net-want"},
		MacAddress:     "aa:bb:cc:dd:ee:ff",
		TxChecksumming: true,
	}
}

// buildWantNet pushes the adapter to net-want when it diverges.
func buildWantNet(actual *client.OpenIaaSNetworkAdapter) *client.UpdateOpenIaasNetworkAdapterRequest {
	if actual.Network.ID == "net-want" {
		return nil
	}
	return &client.UpdateOpenIaasNetworkAdapterRequest{NetworkID: "net-want"}
}

func TestVIFRetryStopsAfterSuccess(t *testing.T) {
	counts := &vifCallCounts{}
	funcs := scriptedVIFFuncs(counts, []*client.OpenIaaSNetworkAdapter{divergedAdapter()}, []error{nil})
	if err := runVIFUpdateWithRetry(context.Background(), "vif-1", funcs, buildWantNet); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if counts.updates != 1 || counts.sleeps != 0 {
		t.Fatalf("updates=%d sleeps=%d, want 1/0 (no extra call after success)", counts.updates, counts.sleeps)
	}
}

func TestVIFRetryRecomputesOrSkipsWhenConverged(t *testing.T) {
	// First attempt fails transiently, but the PATCH actually landed
	// server-side: the second read returns a converged adapter and the
	// rebuilt payload is nil — success WITHOUT re-sending the stale PATCH
	// (which the platform would reject as a static-IP self-conflict).
	counts := &vifCallCounts{}
	funcs := scriptedVIFFuncs(counts,
		[]*client.OpenIaaSNetworkAdapter{divergedAdapter(), convergedAdapter()},
		[]error{errTransientVIF})
	if err := runVIFUpdateWithRetry(context.Background(), "vif-1", funcs, buildWantNet); err != nil {
		t.Fatalf("converged adapter after transient failure must succeed, got: %s", err)
	}
	if counts.updates != 1 {
		t.Fatalf("updates=%d, want exactly 1 (no stale re-PATCH on a converged adapter)", counts.updates)
	}
	if counts.reads != 2 {
		t.Fatalf("reads=%d, want 2 (live state re-read before the retry)", counts.reads)
	}
}

func TestVIFRetryMaxThreeAttemptsTotal(t *testing.T) {
	counts := &vifCallCounts{}
	funcs := scriptedVIFFuncs(counts, []*client.OpenIaaSNetworkAdapter{divergedAdapter()}, []error{errTransientVIF})
	err := runVIFUpdateWithRetry(context.Background(), "vif-1", funcs, buildWantNet)
	if err == nil {
		t.Fatal("an uninterrupted stream of transient failures must eventually fail")
	}
	if counts.updates != maxTransientVIFAttempts {
		t.Fatalf("updates=%d, want %d TOTAL attempts (not 1+%d)", counts.updates, maxTransientVIFAttempts, maxTransientVIFAttempts)
	}
	if counts.sleeps != maxTransientVIFAttempts-1 {
		t.Fatalf("sleeps=%d, want %d", counts.sleeps, maxTransientVIFAttempts-1)
	}
}

func TestVIFRetryDoesNotRetryPermanentFailure(t *testing.T) {
	permanent := errors.New("MAC address is already used by virtual machine vm-2")
	counts := &vifCallCounts{}
	funcs := scriptedVIFFuncs(counts, []*client.OpenIaaSNetworkAdapter{divergedAdapter()}, []error{permanent})
	err := runVIFUpdateWithRetry(context.Background(), "vif-1", funcs, buildWantNet)
	if !errors.Is(err, permanent) {
		t.Fatalf("permanent failure must surface immediately, got: %v", err)
	}
	if counts.updates != 1 || counts.sleeps != 0 {
		t.Fatalf("updates=%d sleeps=%d, want 1/0 (permanent failures are never retried)", counts.updates, counts.sleeps)
	}
}

func TestVIFRetryAdapterGoneIsAnError(t *testing.T) {
	counts := &vifCallCounts{}
	funcs := scriptedVIFFuncs(counts, []*client.OpenIaaSNetworkAdapter{nil}, []error{nil})
	err := runVIFUpdateWithRetry(context.Background(), "vif-1", funcs, buildWantNet)
	if err == nil {
		t.Fatal("a missing adapter must be an explicit error, never a silent skip")
	}
	if counts.updates != 0 {
		t.Fatalf("updates=%d, want 0 (never act on a stale id)", counts.updates)
	}
}

func TestVIFRetryAlreadyConvergedIsANoOp(t *testing.T) {
	counts := &vifCallCounts{}
	funcs := scriptedVIFFuncs(counts, []*client.OpenIaaSNetworkAdapter{convergedAdapter()}, []error{nil})
	if err := runVIFUpdateWithRetry(context.Background(), "vif-1", funcs, buildWantNet); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if counts.updates != 0 {
		t.Fatalf("updates=%d, want 0 (converged adapter produces no PATCH)", counts.updates)
	}
}
