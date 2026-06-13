package provider

import (
	"context"
	"errors"
	"testing"
)

func vmwareLister(ids []string, err error) func(context.Context) ([]string, error) {
	return func(context.Context) ([]string, error) {
		return ids, err
	}
}

// TestConfirmVMwareDeviceLiveness pins the liveness check: it reports presence
// from a strict scoped listing and never concludes a deletion (#281).
func TestConfirmVMwareDeviceLiveness(t *testing.T) {
	ctx := context.Background()

	t.Run("a list error propagates and reports not present", func(t *testing.T) {
		boom := errors.New("boom")
		present, err := confirmVMwareDeviceLiveness(ctx, "d1", vmwareLister(nil, boom))
		if !errors.Is(err, boom) {
			t.Fatalf("the list error must propagate, got %v", err)
		}
		if present {
			t.Fatal("a list error must never report a live device")
		}
	})

	t.Run("an exact id match is present", func(t *testing.T) {
		present, err := confirmVMwareDeviceLiveness(ctx, "d1", vmwareLister([]string{"x", "d1", "y"}, nil))
		if err != nil || !present {
			t.Fatalf("expected present, got present=%v err=%v", present, err)
		}
	})

	t.Run("an absent id is not present", func(t *testing.T) {
		present, err := confirmVMwareDeviceLiveness(ctx, "d1", vmwareLister([]string{"x", "y"}, nil))
		if err != nil || present {
			t.Fatalf("expected absent, got present=%v err=%v", present, err)
		}
	})

	t.Run("an empty listing is not present", func(t *testing.T) {
		present, err := confirmVMwareDeviceLiveness(ctx, "d1", vmwareLister(nil, nil))
		if err != nil || present {
			t.Fatalf("expected absent, got present=%v err=%v", present, err)
		}
	})

	t.Run("substring and superstring ids do not false-match", func(t *testing.T) {
		present, err := confirmVMwareDeviceLiveness(ctx, "d1", vmwareLister([]string{"d12", "d", "d1x"}, nil))
		if err != nil || present {
			t.Fatalf("only an exact id match counts, got present=%v err=%v", present, err)
		}
	})

	t.Run("empty entries are skipped without hiding a present id", func(t *testing.T) {
		present, err := confirmVMwareDeviceLiveness(ctx, "d1", vmwareLister([]string{"", "d1", ""}, nil))
		if err != nil || !present {
			t.Fatalf("a present id must be found despite empty entries, got present=%v err=%v", present, err)
		}
	})

	t.Run("an empty target id never matches a skipped empty entry", func(t *testing.T) {
		present, err := confirmVMwareDeviceLiveness(ctx, "", vmwareLister([]string{"", "x"}, nil))
		if err != nil || present {
			t.Fatalf("an empty id must never match, got present=%v err=%v", present, err)
		}
	})
}

// TestConfirmVMwareDeviceOrKeep pins the nil-read handler. It returns
// diagnostics but never receives the ResourceData, so it is structurally
// incapable of dropping the resource: every inconclusive branch is an error
// that keeps the state, and only a confirmed liveness returns no diagnostics
// (#281).
func TestConfirmVMwareDeviceOrKeep(t *testing.T) {
	ctx := context.Background()

	t.Run("an empty scope id fails closed", func(t *testing.T) {
		diags := confirmVMwareDeviceOrKeep(ctx, "x", "virtual disk", "virtual machine", "", vmwareLister([]string{"x"}, nil))
		if !diags.HasError() {
			t.Fatal("a missing scope id must fail closed, never silently keep going")
		}
	})

	t.Run("a list error fails closed", func(t *testing.T) {
		diags := confirmVMwareDeviceOrKeep(ctx, "x", "virtual disk", "virtual machine", "vm-1", vmwareLister(nil, errors.New("boom")))
		if !diags.HasError() {
			t.Fatal("a failing listing must fail closed, never drop")
		}
	})

	t.Run("an unconfirmed absence fails closed (never drops)", func(t *testing.T) {
		diags := confirmVMwareDeviceOrKeep(ctx, "x", "virtual disk", "virtual machine", "vm-1", vmwareLister([]string{"other"}, nil))
		if !diags.HasError() {
			t.Fatal("an unconfirmed absence must fail closed, never auto-remove the resource")
		}
	})

	t.Run("a confirmed liveness returns no diagnostics", func(t *testing.T) {
		diags := confirmVMwareDeviceOrKeep(ctx, "x", "virtual disk", "virtual machine", "vm-1", vmwareLister([]string{"x"}, nil))
		if diags.HasError() {
			t.Fatalf("a confirmed liveness must not error: %v", diags)
		}
		if len(diags) != 0 {
			t.Fatalf("a confirmed liveness must return no diagnostics, got %v", diags)
		}
	})
}
