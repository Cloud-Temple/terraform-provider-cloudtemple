package provider

import "sync"

// keyedMutex serializes work per string key WITHIN THIS PROCESS. Terraform
// applies sibling resources concurrently (default parallelism 10), so several
// VM-scoped writes (patch / resize / power / delete, and the future disk / nic /
// snapshot sub-resources) can target the SAME virtual machine at once. The
// upstream API has no cross-request locking, and SDKv2 removed the old
// helper/mutexkv, so this minimal in-process lock keeps concurrent writes to one
// VM from racing (E0-8, #416). It is intentionally process-local: it does not,
// and cannot, coordinate across separate `terraform apply` runs.
type keyedMutex struct {
	mu sync.Mutex
	m  map[string]*sync.Mutex
}

func newKeyedMutex() *keyedMutex {
	return &keyedMutex{m: make(map[string]*sync.Mutex)}
}

// lock acquires the per-key lock and returns the matching unlock function. A
// zero key is locked like any other (it just serializes those callers). Usage:
//
//	unlock := mu.lock(id)
//	defer unlock()
func (k *keyedMutex) lock(key string) func() {
	k.mu.Lock()
	lock, ok := k.m[key]
	if !ok {
		lock = &sync.Mutex{}
		k.m[key] = lock
	}
	k.mu.Unlock()

	lock.Lock()
	return lock.Unlock
}

// publicCloudVMInstanceMutex serializes VM-scoped writes keyed by the VM id.
var publicCloudVMInstanceMutex = newKeyedMutex()
