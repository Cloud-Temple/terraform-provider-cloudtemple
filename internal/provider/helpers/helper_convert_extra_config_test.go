package helpers

import (
	"testing"
)

// TestConvertExtraConfigValueStrictParsing pins the strict, key-aware parsing
// of VM extra_config values. The contract: boolean keys accept only
// TRUE/true/FALSE/false (anything else is an error, never a silent default),
// the numeric key parses to an int (a non-numeric string is an error), and any
// other key is passed through as a string. A regression that defaults an
// invalid boolean to false, or that swallows a parse error, would push a wrong
// value to the VMware API — exactly what strict parsing exists to prevent.
func TestConvertExtraConfigValueStrictParsing(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		want    interface{}
		wantErr bool
	}{
		// boolean keys -----------------------------------------------------
		{"enableUUID TRUE", "disk.enableUUID", "TRUE", true, false},
		{"enableUUID true", "disk.enableUUID", "true", true, false},
		{"enableUUID FALSE", "disk.enableUUID", "FALSE", false, false},
		{"enableUUID false", "disk.enableUUID", "false", false, false},
		{"stealclock TRUE", "stealclock.enable", "TRUE", true, false},
		{"use64BitMMIO FALSE", "pciPassthru.use64BitMMIO", "FALSE", false, false},
		// invalid boolean MUST error, not default to false
		{"enableUUID invalid", "disk.enableUUID", "yes", nil, true},
		{"enableUUID empty", "disk.enableUUID", "", nil, true},
		{"enableUUID 1", "disk.enableUUID", "1", nil, true},

		// numeric key ------------------------------------------------------
		{"mmio size 128", "pciPassthru.64bitMMioSizeGB", "128", 128, false},
		{"mmio size 0", "pciPassthru.64bitMMioSizeGB", "0", 0, false},
		{"mmio size non-numeric", "pciPassthru.64bitMMioSizeGB", "big", nil, true},
		{"mmio size empty", "pciPassthru.64bitMMioSizeGB", "", nil, true},

		// default string passthrough --------------------------------------
		{"arbitrary key", "guestinfo.foo", "bar", "bar", false},
		{"arbitrary key empty value", "guestinfo.foo", "", "", false},
		// a TRUE value under a NON-boolean key stays a string (not coerced)
		{"true under string key", "guestinfo.flag", "TRUE", "TRUE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertExtraConfigValue(tt.key, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ConvertExtraConfigValue(%q, %q) = %#v, want error", tt.key, tt.value, got)
				}
				if got != nil {
					t.Errorf("on error the value must be nil, got %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ConvertExtraConfigValue(%q, %q) unexpected error: %v", tt.key, tt.value, err)
			}
			if got != tt.want {
				t.Errorf("ConvertExtraConfigValue(%q, %q) = %#v (%T), want %#v (%T)", tt.key, tt.value, got, got, tt.want, tt.want)
			}
		})
	}
}
