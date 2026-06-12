package provider

import (
	"fmt"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// optionalComputedBooleanGuards is the registry of every Optional+Computed
// boolean attribute of the provider resources. This list is DELIBERATELY
// annoying to maintain: an Optional+Computed boolean read from d.Get cannot
// distinguish an explicit user choice from a value inherited from the live
// API, so pushing it into a PATCH without an explicit raw-config guard
// silently mutates production resources (#246 class, #267).
//
// Before adding an entry here, either:
//   - gate the attribute on d.GetRawConfig() evidence before any API write
//     (pattern: osAdapterTxConfigured / openIaasVMDesiredPropertiesFromResourceData), or
//   - document below why the attribute never reaches an API request.
var optionalComputedBooleanGuards = map[string]string{
	"cloudtemple_compute_iaas_opensource_virtual_machine.secure_boot":                        "raw-config gated in openIaasVMDesiredPropertiesFromResourceData (#267)",
	"cloudtemple_compute_iaas_opensource_virtual_machine.os_disk.connected":                  "UNGATED-LEGACY: reconciled against live state in handleUpdateOSDevices; connect/disconnect paths use live VBD evidence",
	"cloudtemple_compute_iaas_opensource_virtual_machine.os_network_adapter.attached":        "deprecated attribute, never written from the VM path",
	"cloudtemple_compute_iaas_opensource_virtual_machine.os_network_adapter.tx_checksumming": "raw-config gated through osAdapterTxConfigured (map[string]*bool)",
	"cloudtemple_compute_iaas_opensource_network_adapter.tx_checksumming":                    "raw-config gated in openIaasNetworkAdapterUpdate (txConfigured)",
	"cloudtemple_compute_virtual_machine.boot_options.enter_bios_setup":                      "raw-config gated in buildVMwareBootOptionsFromRaw (Lot D)",
	"cloudtemple_compute_virtual_machine.boot_options.boot_retry_enabled":                    "raw-config gated in buildVMwareBootOptionsFromRaw (Lot D)",
	"cloudtemple_compute_virtual_machine.boot_options.efi_secure_boot_enabled":               "raw-config gated in buildVMwareBootOptionsFromRaw (Lot D)",
	"cloudtemple_compute_virtual_machine.os_network_adapter.auto_connect":                    "UNGATED-LEGACY: write guarded by per-index HasChange in updateVirtualMachine; merged value still pushed when any adapter field changes — full raw-config gating tracked in the #264 plan",
	"cloudtemple_compute_virtual_machine.os_network_adapter.connected":                       "UNGATED-LEGACY: connect/disconnect guarded by per-index HasChange on the attribute itself — full raw-config gating tracked in the #264 plan",
}

// collectOptionalComputedBooleans walks a schema map recursively and
// returns the dotted paths of every Optional+Computed TypeBool attribute.
func collectOptionalComputedBooleans(prefix string, s map[string]*schema.Schema, found map[string]bool) {
	for name, attr := range s {
		path := fmt.Sprintf("%s.%s", prefix, name)
		if attr.Type == schema.TypeBool && attr.Optional && attr.Computed {
			found[path] = true
		}
		if nested, ok := attr.Elem.(*schema.Resource); ok {
			collectOptionalComputedBooleans(path, nested.Schema, found)
		}
	}
}

func TestOptionalComputedBooleansHavePatchGuards(t *testing.T) {
	p := New("test")()

	found := map[string]bool{}
	for resourceName, resource := range p.ResourcesMap {
		collectOptionalComputedBooleans(resourceName, resource.Schema, found)
	}

	var paths []string
	for path := range found {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		if _, ok := optionalComputedBooleanGuards[path]; !ok {
			t.Errorf("Optional+Computed boolean %q is not in the guard registry: gate it on raw-config evidence before any API write, or document why it never reaches a request (see #267)", path)
		}
	}
	for path := range optionalComputedBooleanGuards {
		if !found[path] {
			t.Errorf("guard registry entry %q no longer matches any schema attribute: remove or fix it", path)
		}
	}
}
