package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	gib10 = 10737418240 // 10 GiB
	gib20 = 21474836480 // 20 GiB
	gib32 = 34359738368 // 32 GiB
	gib64 = 68719476736 // 64 GiB
)

// Test_394_OsDiskCapacityShrink couvre la détection pure du shrink de disque.
func Test_394_OsDiskCapacityShrink(t *testing.T) {
	cases := []struct {
		name      string
		live, req int
		want      bool
	}{
		{"shrink (32->10)", gib32, gib10, true},
		{"grow (10->32)", gib10, gib32, false},
		{"égal", gib32, gib32, false},
		{"requested omis (0) => pas un shrink", gib32, 0, false},
		{"live inconnu (0) => pas un shrink", 0, gib10, false},
		{"requested négatif => pas traité ici (rejeté par ValidateFunc en amont)", gib32, -1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := osDiskCapacityShrink(tc.live, tc.req); got != tc.want {
				t.Fatalf("osDiskCapacityShrink(%d,%d) = %v, want %v", tc.live, tc.req, got, tc.want)
			}
		})
	}
}

// Test_394_PlanGuard exerce le garde-fou de plan via Resource.Diff hors-ligne :
// un shrink explicite d'os_disk.capacity est refusé ; grow / égal / capacité omise
// passent ; sur plusieurs disques seul celui qui rétrécit est refusé.
func Test_394_PlanGuard(t *testing.T) {
	res := resourceVirtualMachine()
	diffErr := func(state map[string]string, cfg map[string]interface{}) error {
		st := &terraform.InstanceState{ID: "vm-1", Attributes: state}
		_, err := res.Diff(context.Background(), st, terraform.NewResourceConfigRaw(cfg), nil)
		return err
	}
	stateDisks := func(caps ...string) map[string]string {
		m := map[string]string{"os_disk.#": itoa(len(caps))}
		for i, c := range caps {
			m[fmt.Sprintf("os_disk.%d.capacity", i)] = c
			m[fmt.Sprintf("os_disk.%d.id", i)] = fmt.Sprintf("disk-%d", i)
			m[fmt.Sprintf("os_disk.%d.disk_mode", i)] = "persistent"
		}
		return m
	}
	cfg := func(disks ...map[string]interface{}) map[string]interface{} {
		blocks := make([]interface{}, len(disks))
		for i, d := range disks {
			blocks[i] = d
		}
		return map[string]interface{}{
			"datacenter_id":   "dc",
			"host_cluster_id": "hc",
			"os_disk":         blocks,
		}
	}

	t.Run("shrink explicite (32->10) => REFUS", func(t *testing.T) {
		err := diffErr(stateDisks("34359738368"), cfg(map[string]interface{}{"capacity": gib10}))
		if err == nil || !strings.Contains(err.Error(), "cannot shrink os_disk[0]") {
			t.Fatalf("attendu refus de shrink, obtenu: %v", err)
		}
	})
	t.Run("grow (32->64) => OK", func(t *testing.T) {
		if err := diffErr(stateDisks("34359738368"), cfg(map[string]interface{}{"capacity": gib64})); err != nil {
			t.Fatalf("attendu pas d'erreur, obtenu: %v", err)
		}
	})
	t.Run("égal (32->32) => OK", func(t *testing.T) {
		if err := diffErr(stateDisks("34359738368"), cfg(map[string]interface{}{"capacity": gib32})); err != nil {
			t.Fatalf("attendu pas d'erreur, obtenu: %v", err)
		}
	})
	t.Run("capacity omise => OK (adopte le live)", func(t *testing.T) {
		if err := diffErr(stateDisks("34359738368"), cfg(map[string]interface{}{"disk_mode": "persistent"})); err != nil {
			t.Fatalf("attendu pas d'erreur, obtenu: %v", err)
		}
	})
	t.Run("2 disques, seul le 2e rétrécit => REFUS sur [1]", func(t *testing.T) {
		err := diffErr(stateDisks("34359738368", "21474836480"),
			cfg(map[string]interface{}{"capacity": gib32}, map[string]interface{}{"capacity": gib10}))
		if err == nil || !strings.Contains(err.Error(), "os_disk[1]") {
			t.Fatalf("attendu refus sur os_disk[1], obtenu: %v", err)
		}
	})
}
