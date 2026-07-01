package provider

import (
	"testing"
	"time"
)

// TestKeyedMutexSerializesSameKey pins that two lock() calls on the SAME key are
// serialized: the second blocks until the first is unlocked. This is the E0-8
// guarantee that concurrent writes to one VM never race.
func TestKeyedMutexSerializesSameKey(t *testing.T) {
	km := newKeyedMutex()
	unlock := km.lock("vm-1")

	acquired := make(chan struct{})
	go func() {
		u2 := km.lock("vm-1")
		close(acquired)
		u2()
	}()

	select {
	case <-acquired:
		t.Fatal("a second lock on the same key must block while the first is held")
	case <-time.After(50 * time.Millisecond):
		// expected: still blocked while the first holder has not unlocked.
	}

	unlock()

	select {
	case <-acquired:
		// expected: the second lock proceeds once the first is released.
	case <-time.After(2 * time.Second):
		t.Fatal("the second lock must proceed after the first unlocks")
	}
}

// TestKeyedMutexDifferentKeysDoNotBlock pins that locks on DIFFERENT keys are
// independent: writes to distinct VMs run concurrently.
func TestKeyedMutexDifferentKeysDoNotBlock(t *testing.T) {
	km := newKeyedMutex()
	unlock := km.lock("vm-1")
	defer unlock()

	done := make(chan struct{})
	go func() {
		u := km.lock("vm-2")
		u()
		close(done)
	}()

	select {
	case <-done:
		// expected: a different key is not blocked by vm-1's lock.
	case <-time.After(2 * time.Second):
		t.Fatal("locks on different keys must not block each other")
	}
}
