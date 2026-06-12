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
