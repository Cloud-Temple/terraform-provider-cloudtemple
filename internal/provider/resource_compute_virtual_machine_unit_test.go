package provider

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

var bootOptionsRawType = cty.Object(map[string]cty.Type{
	"boot_delay":              cty.Number,
	"boot_retry_delay":        cty.Number,
	"boot_retry_enabled":      cty.Bool,
	"enter_bios_setup":        cty.Bool,
	"firmware":                cty.String,
	"efi_secure_boot_enabled": cty.Bool,
})

func TestBuildVMwareBootOptionsFromRaw(t *testing.T) {
	merged := map[string]interface{}{
		"boot_delay":              5,
		"boot_retry_delay":        10,
		"boot_retry_enabled":      true, // merged from live, NOT configured
		"enter_bios_setup":        true, // merged from live, NOT configured
		"firmware":                "EFI",
		"efi_secure_boot_enabled": true, // merged from live, NOT configured
	}

	t.Run("unconfigured booleans are omitted even when merged true", func(t *testing.T) {
		rawBlock := cty.ObjectVal(map[string]cty.Value{
			"boot_delay":              cty.NumberIntVal(5),
			"boot_retry_delay":        cty.NumberIntVal(10),
			"boot_retry_enabled":      cty.NullVal(cty.Bool),
			"enter_bios_setup":        cty.NullVal(cty.Bool),
			"firmware":                cty.StringVal("EFI"),
			"efi_secure_boot_enabled": cty.NullVal(cty.Bool),
		})
		opts := buildVMwareBootOptionsFromRaw(rawBlock, merged)
		if opts.BootRetryEnabled != nil || opts.EnterBIOSSetup != nil || opts.EFISecureBootEnabled != nil {
			t.Fatalf("unconfigured booleans must be omitted: %+v", opts)
		}
		if opts.BootDelay != 5 || opts.BootRetryDelay != 10 || opts.Firmware != "efi" {
			t.Fatalf("non-boolean fields not carried: %+v", opts)
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
