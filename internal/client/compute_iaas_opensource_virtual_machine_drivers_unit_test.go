package client

import (
	"context"
	"errors"
	"testing"
	"time"
)

// These tests pin OpenIaaSVirtualMachineClient.WaitForDrivers via the
// waitForDrivers seam, driving ticks through an injected channel so there is no
// real 5s wait. They lock the exact behavior the live method relies on (best-effort
// on timeout, parent-vs-timeout ctx, error message format with %s not %w).

func detectedDriverVM() *OpenIaaSVirtualMachine {
	vm := &OpenIaaSVirtualMachine{ID: "vm-1"}
	vm.PVDrivers.Detected = true
	return vm
}

func undetectedDriverVM() *OpenIaaSVirtualMachine {
	return &OpenIaaSVirtualMachine{ID: "vm-1"} // PVDrivers.Detected == false
}

type driverResult struct {
	vm  *OpenIaaSVirtualMachine
	err error
}

func TestWaitForDriversTimeoutZeroSkipsWithParentCtx(t *testing.T) {
	tickerCalled := false
	newTicker := func(d time.Duration) (<-chan time.Time, func()) {
		tickerCalled = true
		return nil, func() {}
	}
	var gotCtx context.Context
	read := func(ctx context.Context) (*OpenIaaSVirtualMachine, error) {
		gotCtx = ctx
		return detectedDriverVM(), nil
	}

	vm, err := waitForDrivers(context.Background(), "vm-1", 0, newTicker, read, nil)
	if err != nil || vm == nil {
		t.Fatalf("timeout==0 must do a single read and return it, got err=%v", err)
	}
	if tickerCalled {
		t.Fatal("timeout==0 must NOT create a ticker")
	}
	if _, ok := gotCtx.Deadline(); ok {
		t.Fatal("timeout==0 must read with the PARENT ctx (no deadline), not a timeout ctx")
	}
}

func TestWaitForDriversTickerIsFiveSecondsAndReadUsesTimeoutCtx(t *testing.T) {
	tickCh := make(chan time.Time, 1)
	var gotDur time.Duration
	newTicker := func(d time.Duration) (<-chan time.Time, func()) {
		gotDur = d
		return tickCh, func() {}
	}
	var hadDeadline bool
	read := func(ctx context.Context) (*OpenIaaSVirtualMachine, error) {
		_, hadDeadline = ctx.Deadline()
		return detectedDriverVM(), nil // detected => returns, terminating the loop
	}

	resCh := make(chan driverResult, 1)
	go func() {
		vm, err := waitForDrivers(context.Background(), "vm-1", time.Hour, newTicker, read, nil)
		resCh <- driverResult{vm, err}
	}()
	tickCh <- time.Now()
	res := <-resCh

	if res.err != nil || res.vm == nil {
		t.Fatalf("a detected VM on a tick must return it, got err=%v", res.err)
	}
	if gotDur != 5*time.Second {
		t.Fatalf("ticker interval=%s, want 5s", gotDur)
	}
	if !hadDeadline {
		t.Fatal("reads while waiting must use the TIMEOUT ctx (with a deadline), not the parent")
	}
}

func TestWaitForDriversTimeoutAfterReadIsBestEffort(t *testing.T) {
	tickCh := make(chan time.Time, 1)
	newTicker := func(d time.Duration) (<-chan time.Time, func()) { return tickCh, func() {} }
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	read := func(c context.Context) (*OpenIaaSVirtualMachine, error) {
		calls++
		cancel() // after this read, the timeout ctx is Done; the loop hits the timeout case
		return undetectedDriverVM(), nil
	}

	resCh := make(chan driverResult, 1)
	go func() {
		vm, err := waitForDrivers(ctx, "vm-1", time.Hour, newTicker, read, nil)
		resCh <- driverResult{vm, err}
	}()
	tickCh <- time.Now()
	res := <-resCh

	if res.err != nil {
		t.Fatalf("a timeout AFTER a non-detected read must be best-effort (nil error), got %v", res.err)
	}
	if res.vm == nil {
		t.Fatal("the timeout path must return the last seen VM (lastVM), not nil")
	}
	if calls != 1 {
		t.Fatalf("calls=%d, want 1", calls)
	}
}

func TestWaitForDriversTimeoutBeforeFirstTickReadsNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled: the timeout ctx is Done before any tick
	tickCh := make(chan time.Time, 1)
	calls := 0
	read := func(c context.Context) (*OpenIaaSVirtualMachine, error) {
		calls++
		return detectedDriverVM(), nil
	}
	newTicker := func(d time.Duration) (<-chan time.Time, func()) { return tickCh, func() {} }

	vm, err := waitForDrivers(ctx, "vm-1", time.Hour, newTicker, read, nil)
	if err != nil {
		t.Fatalf("a timeout before the first tick must return (nil, nil), got err=%v", err)
	}
	if vm != nil {
		t.Fatalf("no read happened, so there is no lastVM; got %v", vm)
	}
	if calls != 0 {
		t.Fatalf("calls=%d, want 0 (timeout fired before any tick)", calls)
	}
}

func TestWaitForDriversReadErrorIsWrappedNotUnwrappable(t *testing.T) {
	tickCh := make(chan time.Time, 1)
	newTicker := func(d time.Duration) (<-chan time.Time, func()) { return tickCh, func() {} }
	sentinel := errors.New("read boom")
	read := func(c context.Context) (*OpenIaaSVirtualMachine, error) {
		return nil, sentinel
	}

	resCh := make(chan driverResult, 1)
	go func() {
		vm, err := waitForDrivers(context.Background(), "vm-1", time.Hour, newTicker, read, nil)
		resCh <- driverResult{vm, err}
	}()
	tickCh <- time.Now()
	res := <-resCh

	if res.vm != nil {
		t.Fatalf("a read error must return a nil VM, got %v", res.vm)
	}
	// Exact message equality pins the full current format (not just a substring).
	wantMsg := `[WAITER] failed to read virtual machine "vm-1" while waiting for drivers: read boom`
	if res.err == nil || res.err.Error() != wantMsg {
		t.Fatalf("read error message must be exactly %q, got %v", wantMsg, res.err)
	}
	// Current behavior formats with a non-wrapping verb (not an error-wrapping one):
	// the original error is not reachable. Pinning this kills both a raw-err return
	// and an error-wrapping "improvement" slipping in under a test PR.
	if errors.Is(res.err, sentinel) {
		t.Fatal("the read error must NOT be wrapped (current non-wrapping semantics); errors.Is must be false")
	}
}

func TestWaitForDriversNilVMIsNotFound(t *testing.T) {
	tickCh := make(chan time.Time, 1)
	newTicker := func(d time.Duration) (<-chan time.Time, func()) { return tickCh, func() {} }
	read := func(c context.Context) (*OpenIaaSVirtualMachine, error) {
		return nil, nil
	}

	resCh := make(chan driverResult, 1)
	go func() {
		vm, err := waitForDrivers(context.Background(), "vm-1", time.Hour, newTicker, read, nil)
		resCh <- driverResult{vm, err}
	}()
	tickCh <- time.Now()
	res := <-resCh

	if res.vm != nil {
		t.Fatalf("a nil VM must return a nil result, got %v", res.vm)
	}
	wantMsg := `[WAITER] the virtual machine "vm-1" could not be found`
	if res.err == nil || res.err.Error() != wantMsg {
		t.Fatalf("a nil VM must return exactly %q, got %v", wantMsg, res.err)
	}
}
