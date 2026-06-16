package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func TestBuildVMwareBootOptionsFromRaw(t *testing.T) {
	merged := map[string]interface{}{
		"boot_delay":              5,
		"boot_retry_delay":        10,
		"boot_retry_enabled":      true, // merged from live, NOT configured
		"enter_bios_setup":        true, // merged from live, NOT configured
		"firmware":                "EFI",
		"efi_secure_boot_enabled": true, // merged from live, NOT configured
	}

	t.Run("unconfigured attributes are omitted even when merged non-zero", func(t *testing.T) {
		rawBlock := cty.ObjectVal(map[string]cty.Value{
			"boot_delay":              cty.NumberIntVal(5),
			"boot_retry_delay":        cty.NullVal(cty.Number),
			"boot_retry_enabled":      cty.NullVal(cty.Bool),
			"enter_bios_setup":        cty.NullVal(cty.Bool),
			"firmware":                cty.StringVal("EFI"),
			"efi_secure_boot_enabled": cty.NullVal(cty.Bool),
		})
		opts := buildVMwareBootOptionsFromRaw(rawBlock, merged)
		if opts.BootRetryEnabled != nil || opts.EnterBIOSSetup != nil || opts.EFISecureBootEnabled != nil {
			t.Fatalf("unconfigured booleans must be omitted: %+v", opts)
		}
		if opts.BootRetryDelay != nil {
			t.Fatalf("unconfigured boot_retry_delay must be omitted (FF-4, no zero push): %+v", opts)
		}
		if opts.BootDelay == nil || *opts.BootDelay != 5 || opts.Firmware != "efi" {
			t.Fatalf("configured attributes not carried: %+v", opts)
		}
	})

	t.Run("a block with no configured attribute returns nil (no empty bootOptions sent)", func(t *testing.T) {
		rawBlock := cty.ObjectVal(map[string]cty.Value{
			"boot_delay":              cty.NullVal(cty.Number),
			"boot_retry_delay":        cty.NullVal(cty.Number),
			"boot_retry_enabled":      cty.NullVal(cty.Bool),
			"enter_bios_setup":        cty.NullVal(cty.Bool),
			"firmware":                cty.NullVal(cty.String),
			"efi_secure_boot_enabled": cty.NullVal(cty.Bool),
		})
		if opts := buildVMwareBootOptionsFromRaw(rawBlock, merged); opts != nil {
			t.Fatalf("an unconfigured block must carry no payload: %+v", opts)
		}
	})

	t.Run("explicit zero boot_delay stays expressible", func(t *testing.T) {
		rawBlock := cty.ObjectVal(map[string]cty.Value{
			"boot_delay":              cty.NumberIntVal(0),
			"boot_retry_delay":        cty.NullVal(cty.Number),
			"boot_retry_enabled":      cty.NullVal(cty.Bool),
			"enter_bios_setup":        cty.NullVal(cty.Bool),
			"firmware":                cty.NullVal(cty.String),
			"efi_secure_boot_enabled": cty.NullVal(cty.Bool),
		})
		opts := buildVMwareBootOptionsFromRaw(rawBlock, merged)
		if opts.BootDelay == nil || *opts.BootDelay != 0 {
			t.Fatalf("explicit zero boot_delay not carried: %+v", opts)
		}
		if opts.Firmware != "" {
			t.Fatalf("unconfigured firmware must be omitted: %+v", opts)
		}
	})

	t.Run("explicit false is carried", func(t *testing.T) {
		rawBlock := cty.ObjectVal(map[string]cty.Value{
			"boot_delay":              cty.NumberIntVal(5),
			"boot_retry_delay":        cty.NumberIntVal(10),
			"boot_retry_enabled":      cty.False,
			"enter_bios_setup":        cty.NullVal(cty.Bool),
			"firmware":                cty.StringVal("EFI"),
			"efi_secure_boot_enabled": cty.False,
		})
		opts := buildVMwareBootOptionsFromRaw(rawBlock, merged)
		if opts.BootRetryDelay == nil || *opts.BootRetryDelay != 10 {
			t.Fatalf("configured boot_retry_delay not carried: %+v", opts)
		}
		if opts.BootRetryEnabled == nil || *opts.BootRetryEnabled != false {
			t.Fatalf("explicit false boot_retry_enabled not carried: %+v", opts)
		}
		if opts.EFISecureBootEnabled == nil || *opts.EFISecureBootEnabled != false {
			t.Fatalf("explicit false efi_secure_boot_enabled not carried: %+v", opts)
		}
		if opts.EnterBIOSSetup != nil {
			t.Fatalf("unconfigured enter_bios_setup must stay omitted: %+v", opts)
		}
	})
}

// TestMigrateVirtualMachineStateV0toV1 pins the only StateUpgrader in the
// provider (#293, S7). The V0 schema stored extra_config as a list of
// {key,value} objects; V1 stores it as a map. The migration must convert the
// list to the map, but must NEVER panic, drop a valid pair, invent a key, or
// corrupt sibling state keys on an N-1 -> N upgrade — the catastrophic class
// this state-safety program guards against. Non-complacent: each case reds out
// if the conversion is dropped, mishandles already-migrated / absent / nil /
// malformed input, changes the duplicate-key resolution, or loses other keys.
func TestMigrateVirtualMachineStateV0toV1(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "v0 list is converted to the v1 map",
			in: map[string]interface{}{
				"extra_config": []interface{}{
					map[string]interface{}{"key": "a", "value": "1"},
					map[string]interface{}{"key": "b", "value": "2"},
				},
			},
			want: map[string]interface{}{
				"extra_config": map[string]interface{}{"a": "1", "b": "2"},
			},
		},
		{
			name: "an already-migrated map is left unchanged (idempotent)",
			in: map[string]interface{}{
				"extra_config": map[string]interface{}{"a": "1"},
			},
			want: map[string]interface{}{
				"extra_config": map[string]interface{}{"a": "1"},
			},
		},
		{
			name: "absent extra_config is untouched (no key invented, no panic)",
			in:   map[string]interface{}{"id": "vm-1"},
			want: map[string]interface{}{"id": "vm-1"},
		},
		{
			name: "nil extra_config stays nil (no panic)",
			in:   map[string]interface{}{"extra_config": nil},
			want: map[string]interface{}{"extra_config": nil},
		},
		{
			name: "empty list becomes an empty map",
			in:   map[string]interface{}{"extra_config": []interface{}{}},
			want: map[string]interface{}{"extra_config": map[string]interface{}{}},
		},
		{
			name: "malformed items are skipped without dropping valid pairs or panicking",
			in: map[string]interface{}{
				"extra_config": []interface{}{
					map[string]interface{}{"key": "a", "value": "1"}, // valid
					"garbage",                            // not a map
					42,                                   // not a map
					map[string]interface{}{"key": "b"},   // missing value
					map[string]interface{}{"value": "2"}, // missing key
					map[string]interface{}{"key": 123, "value": "x"}, // non-string key
					map[string]interface{}{"key": "c", "value": 456}, // non-string value
					map[string]interface{}{"key": "d", "value": "4"}, // valid
				},
			},
			want: map[string]interface{}{
				"extra_config": map[string]interface{}{"a": "1", "d": "4"},
			},
		},
		{
			name: "duplicate keys: the last value wins",
			in: map[string]interface{}{
				"extra_config": []interface{}{
					map[string]interface{}{"key": "k", "value": "1"},
					map[string]interface{}{"key": "k", "value": "2"},
				},
			},
			want: map[string]interface{}{
				"extra_config": map[string]interface{}{"k": "2"},
			},
		},
		{
			name: "sibling state keys are preserved across the migration",
			in: map[string]interface{}{
				"id":   "vm-1",
				"name": "web",
				"extra_config": []interface{}{
					map[string]interface{}{"key": "a", "value": "1"},
				},
			},
			want: map[string]interface{}{
				"id":           "vm-1",
				"name":         "web",
				"extra_config": map[string]interface{}{"a": "1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := migrateVirtualMachineStateV0toV1(context.Background(), tt.in, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("migration mismatch:\n got  = %#v\n want = %#v", got, tt.want)
			}
		})
	}
}
