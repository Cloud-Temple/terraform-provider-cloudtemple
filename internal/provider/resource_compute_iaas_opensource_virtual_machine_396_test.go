package provider

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// Test_396_ForceReassert verrouille le coeur du fix #396 : quand l'utilisateur a
// explicitement changé un champ de taille (forced) mais que le live API rapporte
// DÉJÀ la valeur désirée (cas "forgé" / matérialisation manquée), le PATCH est
// quand même réémis — alors que buildOpenIaasVMPropertiesPatch seul l'avait
// court-circuité (doctrine #267). Sans forced, le court-circuit est préservé.
func Test_396_ForceReassert(t *testing.T) {
	const fourGiB = 4 * 1024 * 1024 * 1024

	t.Run("forgé: live==desired + forced.Memory => PATCH réémis (le coeur de #396)", func(t *testing.T) {
		live := &client.OpenIaaSVirtualMachine{Memory: fourGiB}
		desired := openIaasVMDesiredProperties{Memory: fourGiB}
		req, changed, needsReboot := buildOpenIaasVMPropertiesPatch(live, desired)
		if req.Memory != 0 || changed {
			t.Fatalf("précondition: le build seul doit court-circuiter (req.Memory=%d changed=%v)", req.Memory, changed)
		}
		changed, needsReboot = forceOpenIaasSizingReassert(req, desired, openIaasVMChangedFields{Memory: true}, changed, needsReboot)
		if req.Memory != fourGiB || !changed || !needsReboot {
			t.Fatalf("#396: attendu req.Memory=%d changed=true needsReboot=true, obtenu req.Memory=%d changed=%v needsReboot=%v", fourGiB, req.Memory, changed, needsReboot)
		}
	})

	t.Run("non forcé: live==desired => court-circuit préservé (#267)", func(t *testing.T) {
		live := &client.OpenIaaSVirtualMachine{Memory: fourGiB}
		desired := openIaasVMDesiredProperties{Memory: fourGiB}
		req, changed, needsReboot := buildOpenIaasVMPropertiesPatch(live, desired)
		changed, needsReboot = forceOpenIaasSizingReassert(req, desired, openIaasVMChangedFields{}, changed, needsReboot)
		if req.Memory != 0 || changed || needsReboot {
			t.Fatalf("attendu aucun PATCH, obtenu req.Memory=%d changed=%v needsReboot=%v", req.Memory, changed, needsReboot)
		}
	})

	t.Run("mix réaliste: build pose cpu (live!=desired), force ajoute memory (forgé) — les deux atterrissent", func(t *testing.T) {
		// build doit poser CPU (2->4) car live!=desired, et SAUTER memory (4G==4G, forgé) ;
		// le force doit ensuite AJOUTER memory sans toucher au CPU déjà posé par le build.
		live := &client.OpenIaaSVirtualMachine{CPU: 2, Memory: fourGiB}
		desired := openIaasVMDesiredProperties{CPU: 4, Memory: fourGiB}
		req, changed, needsReboot := buildOpenIaasVMPropertiesPatch(live, desired)
		if req.CPU != 4 || req.Memory != 0 {
			t.Fatalf("précondition: build pose cpu=4 et saute memory (forgé) ; obtenu cpu=%d memory=%d", req.CPU, req.Memory)
		}
		changed, needsReboot = forceOpenIaasSizingReassert(req, desired, openIaasVMChangedFields{Memory: true}, changed, needsReboot)
		if req.CPU != 4 || req.Memory != fourGiB || !changed || !needsReboot {
			t.Fatalf("attendu cpu=4 (intact) + memory=4G (forcé), obtenu cpu=%d memory=%d", req.CPU, req.Memory)
		}
	})

	t.Run("cpu et num_cores_per_socket forcés (forgés)", func(t *testing.T) {
		cores := 2
		desired := openIaasVMDesiredProperties{CPU: 4, NumCoresPerSocket: &cores}
		req := &client.UpdateOpenIaasVirtualMachineRequest{}
		changed, needsReboot := forceOpenIaasSizingReassert(req, desired, openIaasVMChangedFields{CPU: true, NumCoresPerSocket: true}, false, false)
		if req.CPU != 4 || req.NumCoresPerSocket != 2 || !changed || !needsReboot {
			t.Fatalf("attendu cpu=4 cores=2 réémis, obtenu cpu=%d cores=%d changed=%v", req.CPU, req.NumCoresPerSocket, changed)
		}
	})

	t.Run("forced.Memory mais desired.Memory==0 (omis) => pas de PATCH", func(t *testing.T) {
		req := &client.UpdateOpenIaasVirtualMachineRequest{}
		changed, needsReboot := forceOpenIaasSizingReassert(req, openIaasVMDesiredProperties{Memory: 0}, openIaasVMChangedFields{Memory: true}, false, false)
		if req.Memory != 0 || changed || needsReboot {
			t.Fatalf("attendu aucun PATCH (desired omis), obtenu req.Memory=%d changed=%v", req.Memory, changed)
		}
	})
}

// Test_396_PatchNotMaterialised verrouille le read-back : un champ patché dont la
// valeur live ne correspond pas après l'activité est signalé (fail-closed) ;
// un champ non patché (0) est ignoré ; une correspondance ne signale rien.
func Test_396_PatchNotMaterialised(t *testing.T) {
	const fourGiB = 4 * 1024 * 1024 * 1024
	const twoGiB = 2 * 1024 * 1024 * 1024

	t.Run("memory patchée mais live encore à l'ancienne valeur => écart signalé", func(t *testing.T) {
		req := &client.UpdateOpenIaasVirtualMachineRequest{Memory: fourGiB}
		live := &client.OpenIaaSVirtualMachine{Memory: twoGiB}
		if m := openIaasPatchNotMaterialised(req, live); len(m) != 1 {
			t.Fatalf("attendu 1 écart, obtenu %v", m)
		}
	})
	t.Run("memory matérialisée => rien", func(t *testing.T) {
		req := &client.UpdateOpenIaasVirtualMachineRequest{Memory: fourGiB}
		live := &client.OpenIaaSVirtualMachine{Memory: fourGiB}
		if m := openIaasPatchNotMaterialised(req, live); len(m) != 0 {
			t.Fatalf("attendu 0 écart, obtenu %v", m)
		}
	})
	t.Run("champ non patché (req=0) ignoré même si live diffère", func(t *testing.T) {
		req := &client.UpdateOpenIaasVirtualMachineRequest{Memory: 0}
		live := &client.OpenIaaSVirtualMachine{Memory: twoGiB}
		if m := openIaasPatchNotMaterialised(req, live); len(m) != 0 {
			t.Fatalf("attendu 0 écart (champ non patché), obtenu %v", m)
		}
	})
	t.Run("cpu + cores + memory tous non matérialisés => 3 écarts", func(t *testing.T) {
		req := &client.UpdateOpenIaasVirtualMachineRequest{Memory: fourGiB, CPU: 8, NumCoresPerSocket: 2}
		live := &client.OpenIaaSVirtualMachine{Memory: twoGiB, CPU: 4, NumCoresPerSocket: 1}
		if m := openIaasPatchNotMaterialised(req, live); len(m) != 3 {
			t.Fatalf("attendu 3 écarts, obtenu %v", m)
		}
	})
}

type fakeChangeReader struct {
	isNew   bool
	changes map[string]bool
}

func (f fakeChangeReader) IsNewResource() bool     { return f.isNew }
func (f fakeChangeReader) HasChange(k string) bool { return f.changes[k] }

// Test_396_ForcedFields verrouille le CÂBLAGE du fix (le point que la revue
// multi-agents a signalé comme non testé) : le gate "jamais sur create" et le
// mapping exact des clés HasChange -> openIaasVMChangedFields. Une inversion du
// gate, une typo de clé ("cpus") ou un swap ("cpu" <-> "memory") font échouer
// l'un de ces cas — alors que les tests des helpers feuilles ne les verraient pas.
func Test_396_ForcedFields(t *testing.T) {
	t.Run("create => aucun champ forcé (gate)", func(t *testing.T) {
		got := openIaasVMForcedFields(fakeChangeReader{isNew: true, changes: map[string]bool{"memory": true, "cpu": true, "num_cores_per_socket": true}})
		if got != (openIaasVMChangedFields{}) {
			t.Fatalf("create: attendu zéro (gate), obtenu %+v", got)
		}
	})
	t.Run("update + memory changé => Memory seul (mapping de clé exact)", func(t *testing.T) {
		got := openIaasVMForcedFields(fakeChangeReader{changes: map[string]bool{"memory": true}})
		if got != (openIaasVMChangedFields{Memory: true}) {
			t.Fatalf("attendu {Memory:true} ; une typo/swap de clé casserait ce test ; obtenu %+v", got)
		}
	})
	t.Run("update + cpu changé => CPU seul", func(t *testing.T) {
		got := openIaasVMForcedFields(fakeChangeReader{changes: map[string]bool{"cpu": true}})
		if got != (openIaasVMChangedFields{CPU: true}) {
			t.Fatalf("attendu {CPU:true}, obtenu %+v", got)
		}
	})
	t.Run("update + num_cores_per_socket changé => NumCoresPerSocket seul", func(t *testing.T) {
		got := openIaasVMForcedFields(fakeChangeReader{changes: map[string]bool{"num_cores_per_socket": true}})
		if got != (openIaasVMChangedFields{NumCoresPerSocket: true}) {
			t.Fatalf("attendu {NumCoresPerSocket:true}, obtenu %+v", got)
		}
	})
	t.Run("update sans changement de taille => zéro", func(t *testing.T) {
		got := openIaasVMForcedFields(fakeChangeReader{changes: map[string]bool{"name": true}})
		if got != (openIaasVMChangedFields{}) {
			t.Fatalf("attendu zéro, obtenu %+v", got)
		}
	})
}
