package provider

import "testing"

func TestParseVMScopedID(t *testing.T) {
	vm := "11111111-1111-1111-1111-111111111111"
	child := "22222222-2222-2222-2222-222222222222"

	t.Run("valid round-trip", func(t *testing.T) {
		gotVM, gotChild, err := parseVMScopedID(formatVMScopedID(vm, child))
		if err != nil || gotVM != vm || gotChild != child {
			t.Fatalf("round-trip failed: vm=%q child=%q err=%v", gotVM, gotChild, err)
		}
	})

	bad := map[string]string{
		"empty":          "",
		"single part":    vm,
		"trailing slash": vm + "/",
		"leading slash":  "/" + child,
		"three parts":    vm + "/" + child + "/extra",
		"vm not uuid":    "not-a-uuid/" + child,
		"child not uuid": vm + "/not-a-uuid",
		"both not uuid":  "a/b",
		"empty middle":   vm + "//" + child,
	}
	for name, id := range bad {
		t.Run("rejects "+name, func(t *testing.T) {
			if _, _, err := parseVMScopedID(id); err == nil {
				t.Fatalf("expected error for %q", id)
			}
		})
	}
}
