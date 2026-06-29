package provider

import (
	"context"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Test_395_NoSilentSizingShrink est le test de régression du bug #395 : retirer
// `memory` / `cpu` / `num_cores_per_socket` du HCL d'une VM déjà gérée ne doit PAS
// planifier un shrink silencieux vers les anciens Defaults (32 MiB / 1 vCPU / 1).
// RED sur le code d'origine (Default + non-Computed), GREEN une fois les champs
// passés Optional+Computed. Le diff est calculé hors-ligne via Resource.Diff.
func Test_395_NoSilentSizingShrink(t *testing.T) {
	res := resourceVirtualMachine()
	ctx := context.Background()

	state := &terraform.InstanceState{
		ID: "vm-123",
		Attributes: map[string]string{
			"memory":               "4294967296", // 4 GiB
			"cpu":                  "8",
			"num_cores_per_socket": "2",
		},
	}

	// Scénario 1 — les 3 attributs sont OMIS du HCL : aucun shrink ne doit être planifié.
	cfgOmit := terraform.NewResourceConfigRaw(map[string]interface{}{
		"name":                         "vm",
		"datacenter_id":                "dc",
		"host_cluster_id":              "hc",
		"guest_operating_system_moref": "amazonlinux2_64Guest",
		// memory / cpu / num_cores_per_socket volontairement absents
	})
	diff, err := res.Diff(ctx, state, cfgOmit, nil)
	if err != nil {
		t.Fatalf("Diff (omit) a échoué: %s", err)
	}
	noShrink := func(attr, shrinkTo string) {
		if diff == nil {
			return
		}
		if a, ok := diff.Attributes[attr]; ok && a.New == shrinkTo && a.New != a.Old {
			t.Fatalf("#395: %q est silencieusement réduit (%s -> %s) alors qu'il est omis du HCL", attr, a.Old, a.New)
		}
	}
	noShrink("memory", "33554432")
	noShrink("cpu", "1")
	noShrink("num_cores_per_socket", "1")

	// Scénario 2 (anti-complaisance) — un changement EXPLICITE de memory doit, lui,
	// toujours être détecté (on ne supprime pas tous les diffs de memory).
	cfgChange := terraform.NewResourceConfigRaw(map[string]interface{}{
		"name":                         "vm",
		"datacenter_id":                "dc",
		"host_cluster_id":              "hc",
		"guest_operating_system_moref": "amazonlinux2_64Guest",
		"memory":                       8589934592, // 8 GiB explicites
		"cpu":                          8,
		"num_cores_per_socket":         2,
	})
	diff2, err := res.Diff(ctx, state, cfgChange, nil)
	if err != nil {
		t.Fatalf("Diff (change) a échoué: %s", err)
	}
	a, ok := diff2.Attributes["memory"]
	if !ok || a.Old != "4294967296" || a.New != "8589934592" {
		t.Fatalf("#395: un changement explicite de memory doit être détecté (4 GiB -> 8 GiB); diff obtenu: %+v", a)
	}
}

// Test_395_FromScratchMissingRequired verrouille le garde-fou : memory + cpu sont
// obligatoires UNIQUEMENT en create from-scratch, et le garde-fou fail-OPEN sur
// tout ce qui n'est pas un create from-scratch certain.
func Test_395_FromScratchMissingRequired(t *testing.T) {
	cases := []struct {
		name                                              string
		isCreate, cloneAbs, clAbs, mktAbs, memAbs, cpuAbs bool
		want                                              []string
	}{
		{"from-scratch sans memory ni cpu", true, true, true, true, true, true, []string{"`memory`", "`cpu`"}},
		{"from-scratch sans memory", true, true, true, true, true, false, []string{"`memory`"}},
		{"from-scratch sans cpu", true, true, true, true, false, true, []string{"`cpu`"}},
		{"from-scratch complet (accepté)", true, true, true, true, false, false, nil},
		{"clone présent => pas d'exigence", true, false, true, true, true, true, nil},
		{"content-library présent => pas d'exigence", true, true, false, true, true, true, nil},
		{"marketplace présent => pas d'exigence", true, true, true, false, true, true, nil},
		{"source inconnue au plan (fail-open)", true, false, true, true, true, true, nil},
		{"update (jamais d'exigence)", false, true, true, true, true, true, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := vmFromScratchMissingRequired(tc.isCreate, tc.cloneAbs, tc.clAbs, tc.mktAbs, tc.memAbs, tc.cpuAbs)
			if !equalStrings(got, tc.want) {
				t.Fatalf("attendu %v, obtenu %v", tc.want, got)
			}
		})
	}
}

// Test_395_ResolveVMwareUpdateSizing verrouille le résolveur anti-zéro de l'update.
func Test_395_ResolveVMwareUpdateSizing(t *testing.T) {
	vm := &client.VirtualMachine{Memory: 4294967296, Cpu: 8, NumCoresPerSocket: 2}

	t.Run("toutes valeurs fournies => pas de live nécessaire", func(t *testing.T) {
		m, c, cs, err := resolveVMwareUpdateSizing(2147483648, 4, 1, nil)
		if err != nil || m != 2147483648 || c != 4 || cs != 1 {
			t.Fatalf("got m=%d c=%d cs=%d err=%v", m, c, cs, err)
		}
	})
	t.Run("memory omis => substitué par le live", func(t *testing.T) {
		m, c, cs, err := resolveVMwareUpdateSizing(0, 4, 1, vm)
		if err != nil || m != 4294967296 || c != 4 || cs != 1 {
			t.Fatalf("got m=%d c=%d cs=%d err=%v", m, c, cs, err)
		}
	})
	t.Run("les 3 omis => tous substitués par le live", func(t *testing.T) {
		m, c, cs, err := resolveVMwareUpdateSizing(0, 0, 0, vm)
		if err != nil || m != 4294967296 || c != 8 || cs != 2 {
			t.Fatalf("got m=%d c=%d cs=%d err=%v", m, c, cs, err)
		}
	})
	t.Run("fail-closed: live nil quand une substitution est nécessaire", func(t *testing.T) {
		if _, _, _, err := resolveVMwareUpdateSizing(0, 4, 1, nil); err == nil {
			t.Fatalf("attendu une erreur (live nil), aucune obtenue")
		}
	})
	t.Run("fail-closed: live rapporte aussi 0", func(t *testing.T) {
		zero := &client.VirtualMachine{Memory: 0, Cpu: 8, NumCoresPerSocket: 2}
		if _, _, _, err := resolveVMwareUpdateSizing(0, 4, 1, zero); err == nil {
			t.Fatalf("attendu une erreur (live memory=0), aucune obtenue")
		}
	})
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
