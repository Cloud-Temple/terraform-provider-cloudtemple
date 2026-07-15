package provider

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	g1 = 1073741824 // 1 GiB
	g2 = 2147483648 // 2 GiB
	g4 = 4294967296 // 4 GiB
)

// Test_397_NeedsPowerCycle couvre exhaustivement la décision de power-cycle :
// quelle combinaison taille/flags/power exige d'éteindre la VM avant le PATCH.
func Test_397_NeedsPowerCycle(t *testing.T) {
	base := func() vmwareSizingChange {
		return vmwareSizingChange{
			running: true,
			curMem:  g2, newMem: g2,
			curCPU: 2, newCPU: 2,
			curCores: 1, newCores: 1,
		}
	}
	cases := []struct {
		name string
		mut  func(*vmwareSizingChange)
		want bool
	}{
		{"VM éteinte : jamais de cycle même si tout change", func(c *vmwareSizingChange) {
			c.running = false
			c.newMem = g4
			c.newCPU = 8
			c.newCores = 4
			c.memHotAddChanged = true
		}, false},
		{"rien ne change", func(c *vmwareSizingChange) {}, false},
		{"hausse RAM + hot-add ON", func(c *vmwareSizingChange) { c.newMem = g4; c.curMemHotAdd = true }, false},
		{"hausse RAM + hot-add OFF", func(c *vmwareSizingChange) { c.newMem = g4; c.curMemHotAdd = false }, true},
		{"baisse RAM + hot-add ON (pas de hot-remove RAM)", func(c *vmwareSizingChange) { c.newMem = g1; c.curMemHotAdd = true }, true},
		{"baisse RAM + hot-add OFF", func(c *vmwareSizingChange) { c.newMem = g1; c.curMemHotAdd = false }, true},
		{"RAM omise (newMem=0) => pas de changement", func(c *vmwareSizingChange) { c.newMem = 0 }, false},
		{"hausse CPU + cpu hot-add ON", func(c *vmwareSizingChange) { c.newCPU = 4; c.curCPUHotAdd = true }, false},
		{"hausse CPU + cpu hot-add OFF", func(c *vmwareSizingChange) { c.newCPU = 4; c.curCPUHotAdd = false }, true},
		{"baisse CPU + cpu hot-remove ON", func(c *vmwareSizingChange) { c.newCPU = 1; c.curCPUHotRemove = true }, false},
		{"baisse CPU + cpu hot-remove OFF", func(c *vmwareSizingChange) { c.newCPU = 1; c.curCPUHotRemove = false }, true},
		{"CPU omis (newCPU=0)", func(c *vmwareSizingChange) { c.newCPU = 0 }, false},
		{"changement cores-per-socket => toujours off", func(c *vmwareSizingChange) { c.newCores = 2 }, true},
		{"cores omis (newCores=0)", func(c *vmwareSizingChange) { c.newCores = 0 }, false},
		{"toggle memory_hot_add_enabled", func(c *vmwareSizingChange) { c.memHotAddChanged = true }, true},
		{"toggle cpu_hot_add_enabled", func(c *vmwareSizingChange) { c.cpuHotAddChanged = true }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch := base()
			tc.mut(&ch)
			if got := vmwareNeedsPowerCycle(ch); got != tc.want {
				t.Fatalf("vmwareNeedsPowerCycle = %v, want %v (%+v)", got, tc.want, ch)
			}
		})
	}
}

// Test_397_SizingNeedsPatch verrouille le gate « zéro appel mutateur » : aucune
// divergence vs le live (et aucun champ non-comparable changé) => pas de PATCH.
func Test_397_SizingNeedsPatch(t *testing.T) {
	live := &client.VirtualMachine{
		Memory: g2, Cpu: 2, NumCoresPerSocket: 1,
		MemoryHotAddEnabled: true, CpuHotAddEnabled: false, CpuHotRemoveEnabled: false,
	}
	cases := []struct {
		name                                    string
		mem, cpu, cores                         int
		memHA, cpuHA, cpuHR                     bool
		reservationChanged, exposeChanged, boot bool
		want                                    bool
	}{
		{"tout égal au live, rien d'autre changé => pas de PATCH", g2, 2, 1, true, false, false, false, false, false, false},
		{"flag absent résolu au live (=true) => pas de PATCH", g2, 2, 1, true, false, false, false, false, false, false},
		{"memory diverge", g4, 2, 1, true, false, false, false, false, false, true},
		{"cpu diverge", g2, 4, 1, true, false, false, false, false, false, true},
		{"cores diverge", g2, 2, 2, true, false, false, false, false, false, true},
		{"memory_hot_add explicitement changé", g2, 2, 1, false, false, false, false, false, false, true},
		{"cpu_hot_remove explicitement changé", g2, 2, 1, true, false, true, false, false, false, true},
		{"memory_reservation changé (non comparable au live)", g2, 2, 1, true, false, false, true, false, false, true},
		{"expose changé", g2, 2, 1, true, false, false, false, true, false, true},
		{"boot_options changé", g2, 2, 1, true, false, false, false, false, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := vmwareSizingNeedsPatch(tc.mem, tc.cpu, tc.cores, tc.memHA, tc.cpuHA, tc.cpuHR, live, tc.reservationChanged, tc.exposeChanged, tc.boot)
			if got != tc.want {
				t.Fatalf("vmwareSizingNeedsPatch = %v, want %v", got, tc.want)
			}
		})
	}
}

// Test_397_ResolveBoolFromConfig : explicite => valeur config (+ changed) ; absent
// ou config nulle => valeur live (jamais de faux changement de flag).
func Test_397_ResolveBoolFromConfig(t *testing.T) {
	objWith := func(v cty.Value) cty.Value {
		return cty.ObjectVal(map[string]cty.Value{"f": v})
	}
	t.Run("explicite true, live false => (true, changed)", func(t *testing.T) {
		v, ch := resolveVMwareBoolFromConfig(objWith(cty.BoolVal(true)), "f", false)
		if !v || !ch {
			t.Fatalf("got (%v,%v), want (true,true)", v, ch)
		}
	})
	t.Run("explicite false, live false => (false, unchanged)", func(t *testing.T) {
		v, ch := resolveVMwareBoolFromConfig(objWith(cty.BoolVal(false)), "f", false)
		if v || ch {
			t.Fatalf("got (%v,%v), want (false,false)", v, ch)
		}
	})
	t.Run("absent (null attr), live true => adopte live, unchanged", func(t *testing.T) {
		v, ch := resolveVMwareBoolFromConfig(objWith(cty.NullVal(cty.Bool)), "f", true)
		if !v || ch {
			t.Fatalf("got (%v,%v), want (true,false)", v, ch)
		}
	})
	t.Run("config nulle => adopte live, unchanged", func(t *testing.T) {
		v, ch := resolveVMwareBoolFromConfig(cty.NullVal(cty.Object(map[string]cty.Type{"f": cty.Bool})), "f", true)
		if !v || ch {
			t.Fatalf("got (%v,%v), want (true,false)", v, ch)
		}
	})
}

// Test_397_PlanGuard exerce le garde-fou de plan via Resource.Diff hors-ligne :
// un changement de taille exigeant un redémarrage sur VM allumée est refusé,
// sauf allow_vm_restart=true ou VM mise à l'arrêt ; et jamais bloqué hors de ce cas.
func Test_397_PlanGuard(t *testing.T) {
	res := resourceVirtualMachine()
	diffErr := func(state map[string]string, cfg map[string]interface{}) error {
		st := &terraform.InstanceState{ID: "vm-1", Attributes: state}
		_, err := res.Diff(context.Background(), st, terraform.NewResourceConfigRaw(cfg), nil)
		return err
	}
	runningState := func() map[string]string {
		return map[string]string{
			"memory":                 itoa(g2),
			"cpu":                    "2",
			"num_cores_per_socket":   "1",
			"power_state":            "on",
			"memory_hot_add_enabled": "false",
			"cpu_hot_add_enabled":    "false",
			"cpu_hot_remove_enabled": "false",
		}
	}
	cfgKeep := func(extra map[string]interface{}) map[string]interface{} {
		c := map[string]interface{}{
			"datacenter_id":   "dc",
			"host_cluster_id": "hc",
			"power_state":     "on",
		}
		for k, v := range extra {
			c[k] = v
		}
		return c
	}

	t.Run("hausse RAM, VM on, hot-add off, allow absent => REFUS", func(t *testing.T) {
		err := diffErr(runningState(), cfgKeep(map[string]interface{}{"memory": g4}))
		if err == nil || !strings.Contains(err.Error(), "requires a restart") {
			t.Fatalf("attendu refus 'requires a restart', obtenu: %v", err)
		}
	})
	t.Run("hausse RAM, allow_vm_restart=true => autorisé", func(t *testing.T) {
		if err := diffErr(runningState(), cfgKeep(map[string]interface{}{"memory": g4, "allow_vm_restart": true})); err != nil {
			t.Fatalf("attendu pas d'erreur, obtenu: %v", err)
		}
	})
	t.Run("hausse RAM mais power_state=off => exempté (VM va s'éteindre)", func(t *testing.T) {
		cfg := cfgKeep(map[string]interface{}{"memory": g4})
		cfg["power_state"] = "off"
		if err := diffErr(runningState(), cfg); err != nil {
			t.Fatalf("attendu pas d'erreur, obtenu: %v", err)
		}
	})
	t.Run("RAM inchangée (pas de changement de taille) => pas de refus", func(t *testing.T) {
		if err := diffErr(runningState(), cfgKeep(map[string]interface{}{"memory": g2})); err != nil {
			t.Fatalf("attendu pas d'erreur, obtenu: %v", err)
		}
	})
	t.Run("VM déjà éteinte (state off) => pas de refus", func(t *testing.T) {
		st := runningState()
		st["power_state"] = "off"
		if err := diffErr(st, cfgKeep(map[string]interface{}{"memory": g4})); err != nil {
			t.Fatalf("attendu pas d'erreur, obtenu: %v", err)
		}
	})
	t.Run("power_state omis (défaut off) sur VM running + hausse RAM => exempté", func(t *testing.T) {
		// Omitting power_state lets the schema Default "off" apply: the VM is being
		// powered down, so the resize needs no allow_vm_restart and must not be refused.
		cfg := map[string]interface{}{"datacenter_id": "dc", "host_cluster_id": "hc", "memory": g4}
		if err := diffErr(runningState(), cfg); err != nil {
			t.Fatalf("attendu pas d'erreur (power_state omis = arrêt), obtenu: %v", err)
		}
	})
	t.Run("hausse RAM + hot-add ON => applicable à chaud, pas de refus", func(t *testing.T) {
		st := runningState()
		st["memory_hot_add_enabled"] = "true"
		if err := diffErr(st, cfgKeep(map[string]interface{}{"memory": g4, "memory_hot_add_enabled": true})); err != nil {
			t.Fatalf("attendu pas d'erreur, obtenu: %v", err)
		}
	})
	t.Run("baisse RAM + hot-add ON => refus (pas de hot-remove RAM)", func(t *testing.T) {
		st := runningState()
		st["memory"] = itoa(g4)
		st["memory_hot_add_enabled"] = "true"
		err := diffErr(st, cfgKeep(map[string]interface{}{"memory": g2, "memory_hot_add_enabled": true}))
		if err == nil || !strings.Contains(err.Error(), "requires a restart") {
			t.Fatalf("attendu refus, obtenu: %v", err)
		}
	})
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
