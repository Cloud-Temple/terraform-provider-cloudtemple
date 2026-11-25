package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// The capacity rename (issue #524) is a decode-contract change, so these tests
// pin the DECODE, not a round-trip through a struct literal — a struct literal
// would pass whatever the json plumbing does and prove nothing.
//
// Each affected type is exercised on four payloads:
//   - current spelling only        -> the value decodes (the post-2026-12-09 API)
//   - deprecated spelling only     -> the value decodes (today's production API)
//   - both, in agreement           -> the value decodes
//   - neither                      -> an ERROR, never a silent zero
//
// The mutation each case is meant to catch:
//   - drop the deprecated fallback  -> the "deprecated only" cases go RED
//   - drop the current branch       -> the "current only" cases go RED
//   - restore the old plain struct field (silent case-insensitive decode)
//     -> the "neither" cases go RED, because a missing field would decode to 0
//   - drop the mismatch guard       -> TestCapacityMismatchIsRejected goes RED

// decodeInto is a thin helper making the intent of each case explicit.
func decodeInto(t *testing.T, payload string, target any) error {
	t.Helper()
	return json.Unmarshal([]byte(payload), target)
}

func mustDecode(t *testing.T, payload string, target any) {
	t.Helper()
	if err := decodeInto(t, payload, target); err != nil {
		t.Fatalf("decode %s: %v", payload, err)
	}
}

func requireCapacityError(t *testing.T, payload string, target any, wantKeys ...string) {
	t.Helper()
	err := decodeInto(t, payload, target)
	if err == nil {
		t.Fatalf("a payload carrying neither spelling must fail, got a successful decode of %s", payload)
	}
	// The diagnostic must name both spellings: it is what tells an operator
	// whether their API is too old or too new.
	for _, k := range wantKeys {
		if !strings.Contains(err.Error(), k) {
			t.Fatalf("error must name %q to be actionable, got: %v", k, err)
		}
	}
}

func TestPublicCloudVMDiskCapacityDecode(t *testing.T) {
	t.Run("current spelling", func(t *testing.T) {
		var d PublicCloudVMDisk
		mustDecode(t, `{"id":"d1","position":1,"label":"data","sizeGib":40,"storageType":"st-1","isPrimary":false}`, &d)
		if d.SizeGib != 40 {
			t.Fatalf("SizeGib = %d, want 40", d.SizeGib)
		}
		// The embedded-alias decode must not swallow the sibling fields — this is
		// the failure mode of the `type plain` + shadow-pointer pattern.
		if d.ID != "d1" || d.Position != 1 || d.Label != "data" || d.StorageType != "st-1" || d.IsPrimary {
			t.Fatalf("non-capacity fields lost by the custom decode: %+v", d)
		}
	})
	t.Run("deprecated spelling", func(t *testing.T) {
		var d PublicCloudVMDisk
		mustDecode(t, `{"id":"d1","sizeGb":40}`, &d)
		if d.SizeGib != 40 {
			t.Fatalf("SizeGib = %d, want 40 from the deprecated sizeGb", d.SizeGib)
		}
	})
	t.Run("both in agreement", func(t *testing.T) {
		var d PublicCloudVMDisk
		mustDecode(t, `{"id":"d1","sizeGib":40,"sizeGb":40}`, &d)
		if d.SizeGib != 40 {
			t.Fatalf("SizeGib = %d, want 40", d.SizeGib)
		}
	})
	t.Run("neither is an error, not a zero", func(t *testing.T) {
		var d PublicCloudVMDisk
		requireCapacityError(t, `{"id":"d1","position":1}`, &d, "sizeGib", "sizeGb", "d1")
	})
}

func TestPublicCloudVMFlavorCapacityDecode(t *testing.T) {
	t.Run("current spelling", func(t *testing.T) {
		var f PublicCloudVMFlavor
		mustDecode(t, `{"id":"f1","name":"m1.small","vcpu":2,"ramGib":8}`, &f)
		if f.RamGib != 8 || f.Name != "m1.small" || f.Vcpu != 2 {
			t.Fatalf("bad decode: %+v", f)
		}
	})
	t.Run("deprecated spelling", func(t *testing.T) {
		var f PublicCloudVMFlavor
		mustDecode(t, `{"id":"f1","ramGb":8}`, &f)
		if f.RamGib != 8 {
			t.Fatalf("RamGib = %d, want 8 from the deprecated ramGb", f.RamGib)
		}
	})
	t.Run("neither is an error", func(t *testing.T) {
		var f PublicCloudVMFlavor
		requireCapacityError(t, `{"id":"f1","name":"m1.small"}`, &f, "ramGib", "ramGb")
	})
}

func TestPublicCloudVMImageCapacityDecode(t *testing.T) {
	t.Run("current spelling", func(t *testing.T) {
		var i PublicCloudVMImage
		mustDecode(t, `{"id":"img-1","name":"Rocky","diskSizesGib":[38,64],"imageType":"os"}`, &i)
		if len(i.DiskSizesGib) != 2 || i.DiskSizesGib[0] != 38 || i.DiskSizesGib[1] != 64 {
			t.Fatalf("DiskSizesGib = %v, want [38 64]", i.DiskSizesGib)
		}
		if i.ImageType != "os" || i.Name != "Rocky" {
			t.Fatalf("non-capacity fields lost: %+v", i)
		}
	})
	t.Run("deprecated spelling", func(t *testing.T) {
		var i PublicCloudVMImage
		mustDecode(t, `{"id":"img-1","diskSizesGb":[38]}`, &i)
		if len(i.DiskSizesGib) != 1 || i.DiskSizesGib[0] != 38 {
			t.Fatalf("DiskSizesGib = %v, want [38] from the deprecated diskSizesGb", i.DiskSizesGib)
		}
	})
	t.Run("an empty list the API really sent is preserved", func(t *testing.T) {
		// Presence, not length, distinguishes the two spellings: an image
		// advertising no size is a value, not an absence.
		var i PublicCloudVMImage
		mustDecode(t, `{"id":"img-1","diskSizesGib":[]}`, &i)
		if i.DiskSizesGib == nil || len(i.DiskSizesGib) != 0 {
			t.Fatalf("an explicitly empty list must survive as empty, got %v", i.DiskSizesGib)
		}
	})
	t.Run("neither is an error", func(t *testing.T) {
		var i PublicCloudVMImage
		requireCapacityError(t, `{"id":"img-1","name":"Rocky"}`, &i, "diskSizesGib", "diskSizesGb")
	})
}

func TestPublicCloudVMInstanceFamilyCapacityDecode(t *testing.T) {
	t.Run("current spelling", func(t *testing.T) {
		var f PublicCloudVMInstanceFamily
		mustDecode(t, `{"id":"fam-1","vcpuMin":1,"vcpuMax":16,"ramMinGib":2,"ramMaxGib":128,"skus":[{"name":"vcpu","price":1.5}]}`, &f)
		if f.RamMinGib != 2 || f.RamMaxGib != 128 {
			t.Fatalf("bad ram bounds: %+v", f)
		}
		// The explicit `skus` json tag lives on the embedded alias; the custom
		// decode must not lose it.
		if len(f.Skus) != 1 || f.Skus[0].Name != "vcpu" || f.Skus[0].Price != 1.5 {
			t.Fatalf("tagged skus field lost by the custom decode: %+v", f.Skus)
		}
	})
	t.Run("deprecated spelling", func(t *testing.T) {
		var f PublicCloudVMInstanceFamily
		mustDecode(t, `{"id":"fam-1","ramMinGb":2,"ramMaxGb":128}`, &f)
		if f.RamMinGib != 2 || f.RamMaxGib != 128 {
			t.Fatalf("bad ram bounds from the deprecated spelling: %+v", f)
		}
	})
	t.Run("mixed spellings across the two bounds", func(t *testing.T) {
		// A partial rollout can rename one field before the other; each bound is
		// resolved independently.
		var f PublicCloudVMInstanceFamily
		mustDecode(t, `{"id":"fam-1","ramMinGib":2,"ramMaxGb":128}`, &f)
		if f.RamMinGib != 2 || f.RamMaxGib != 128 {
			t.Fatalf("bad ram bounds on mixed spellings: %+v", f)
		}
	})
	t.Run("one bound missing is an error", func(t *testing.T) {
		var f PublicCloudVMInstanceFamily
		requireCapacityError(t, `{"id":"fam-1","ramMinGib":2}`, &f, "ramMaxGib", "ramMaxGb")
	})
}

func TestPublicCloudVMQuotaCapacityDecode(t *testing.T) {
	t.Run("current spellings", func(t *testing.T) {
		var q PublicCloudVMQuota
		mustDecode(t, `{"vcpuLimit":100,"ramLimitMib":204800,"storageLimitGib":500,"vcpuUsed":7,"ramUsedMib":10240,"storageUsedGib":148}`, &q)
		if q.RamLimitMib != 204800 || q.StorageLimitGib != 500 || q.RamUsedMib != 10240 || q.StorageUsedGib != 148 {
			t.Fatalf("bad decode: %+v", q)
		}
		if q.VcpuLimit != 100 || q.VcpuUsed != 7 {
			t.Fatalf("non-capacity fields lost: %+v", q)
		}
	})
	t.Run("deprecated spellings", func(t *testing.T) {
		var q PublicCloudVMQuota
		mustDecode(t, `{"vcpuLimit":100,"ramLimitMb":204800,"storageLimitGb":500,"ramUsedMb":10240,"storageUsedGb":148}`, &q)
		if q.RamLimitMib != 204800 || q.StorageLimitGib != 500 || q.RamUsedMib != 10240 || q.StorageUsedGib != 148 {
			t.Fatalf("bad decode from the deprecated spellings: %+v", q)
		}
	})
	t.Run("absent usage counters are zero, not an error", func(t *testing.T) {
		// The API contract gives the two usage counters `default: 0` rather than
		// marking them required — unlike the limits, their absence is a value.
		var q PublicCloudVMQuota
		mustDecode(t, `{"vcpuLimit":100,"ramLimitMib":204800,"storageLimitGib":500}`, &q)
		if q.RamUsedMib != 0 || q.StorageUsedGib != 0 {
			t.Fatalf("absent usage must be zero: %+v", q)
		}
	})
	t.Run("an absent LIMIT is an error", func(t *testing.T) {
		var q PublicCloudVMQuota
		requireCapacityError(t, `{"vcpuLimit":100,"storageLimitGib":500}`, &q, "ramLimitMib", "ramLimitMb")
	})
}

func TestPublicCloudVMStorageTypeCapacityDecode(t *testing.T) {
	t.Run("current spellings", func(t *testing.T) {
		var s PublicCloudVMStorageType
		mustDecode(t, `{"id":"st-1","name":"fast","minSizeGib":1,"maxSizeGib":2048,"isAvailable":true,"sku":{"name":"storage","price":0.1}}`, &s)
		if s.MinSizeGib != 1 || s.MaxSizeGib != 2048 {
			t.Fatalf("bad bounds: %+v", s)
		}
		if !s.IsAvailable || s.Sku == nil || s.Sku.Name != "storage" {
			t.Fatalf("non-capacity fields lost by the custom decode: %+v", s)
		}
	})
	t.Run("deprecated spellings", func(t *testing.T) {
		var s PublicCloudVMStorageType
		mustDecode(t, `{"id":"st-1","minSizeGb":1,"maxSizeGb":2048}`, &s)
		if s.MinSizeGib != 1 || s.MaxSizeGib != 2048 {
			t.Fatalf("bad bounds from the deprecated spellings: %+v", s)
		}
	})
	t.Run("neither is an error", func(t *testing.T) {
		var s PublicCloudVMStorageType
		requireCapacityError(t, `{"id":"st-1","name":"fast"}`, &s, "minSizeGib", "minSizeGb")
	})
}

func TestPublicCloudVMInstanceCapacityDecode(t *testing.T) {
	t.Run("current spellings", func(t *testing.T) {
		var i PublicCloudVMInstance
		mustDecode(t, `{"id":"vm-1","name":"web","vcpu":2,"ramGib":4,"disksSizeGib":40,"guestToolsInstalled":true}`, &i)
		if i.RAMGib != 4 || i.DisksSizeGib != 40 {
			t.Fatalf("bad decode: %+v", i)
		}
		if i.Name != "web" || i.VCPU != 2 || !i.GuestToolsInstalled {
			t.Fatalf("non-capacity fields lost: %+v", i)
		}
	})
	t.Run("deprecated spellings", func(t *testing.T) {
		var i PublicCloudVMInstance
		mustDecode(t, `{"id":"vm-1","ramGb":4,"disksSizeGb":40}`, &i)
		if i.RAMGib != 4 || i.DisksSizeGib != 40 {
			t.Fatalf("bad decode from the deprecated spellings: %+v", i)
		}
	})
	t.Run("a nullable nested ref still decodes", func(t *testing.T) {
		var i PublicCloudVMInstance
		mustDecode(t, `{"id":"vm-1","ramGib":4,"disksSizeGib":40,"backupPolicy":null,"image":{"id":"img-1","name":"Rocky"}}`, &i)
		if i.BackupPolicy != nil {
			t.Fatalf("a null backupPolicy must decode to nil, got %+v", i.BackupPolicy)
		}
		if i.Image.ID != "img-1" {
			t.Fatalf("nested ref lost by the custom decode: %+v", i.Image)
		}
	})
	t.Run("neither is an error", func(t *testing.T) {
		var i PublicCloudVMInstance
		requireCapacityError(t, `{"id":"vm-1","name":"web","vcpu":2}`, &i, "ramGib", "ramGb")
	})
}

// TestCapacityMismatchIsRejected pins the guard against a value being REWRITTEN
// somewhere between the API and the provider. The deprecated name is
// contractually an exact alias, so a divergence means a proxy or an intermediate
// version applied a real conversion (the GB<->GiB factor being the dangerous
// one). Picking either value would record a wrong capacity in the state, so the
// read fails instead. Mutation: drop the equality check and this goes RED.
func TestCapacityMismatchIsRejected(t *testing.T) {
	t.Run("scalar", func(t *testing.T) {
		var d PublicCloudVMDisk
		err := decodeInto(t, `{"id":"d1","sizeGib":50,"sizeGb":54}`, &d)
		if err == nil {
			t.Fatalf("diverging spellings must be rejected, decoded to %+v", d)
		}
		for _, want := range []string{"50", "54", "sizeGib", "sizeGb"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error must report %q to be diagnosable, got: %v", want, err)
			}
		}
	})
	t.Run("list", func(t *testing.T) {
		var i PublicCloudVMImage
		if err := decodeInto(t, `{"id":"img-1","diskSizesGib":[38],"diskSizesGb":[41]}`, &i); err == nil {
			t.Fatalf("diverging list spellings must be rejected, decoded to %+v", i.DiskSizesGib)
		}
	})
	t.Run("list of different lengths", func(t *testing.T) {
		var i PublicCloudVMImage
		if err := decodeInto(t, `{"id":"img-1","diskSizesGib":[38],"diskSizesGb":[38,64]}`, &i); err == nil {
			t.Fatalf("list spellings of different lengths must be rejected, decoded to %+v", i.DiskSizesGib)
		}
	})
}

// TestCapacityErrorSurvivesTheListEndpoints proves the guard is not bypassed by
// the wrapped/paginated list decoders: a broken item must fail the whole read
// rather than yield a list of zero-capacity objects.
func TestCapacityErrorSurvivesTheListEndpoints(t *testing.T) {
	ctx := context.Background()

	t.Run("wrapped disk listing", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"vmId":"vm-1","total":1,"disks":[{"id":"d1","position":1}]}`))
		})
		disks, err := c.PublicCloudVM().Disk().List(ctx, "vm-1")
		if err == nil {
			t.Fatalf("a disk with no capacity must fail the listing, got %d disks", len(disks))
		}
	})

	t.Run("bare storage-type listing", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"storageTypes":[{"id":"st-1","name":"fast","minSizeGib":1,"maxSizeGib":10},{"id":"st-2","name":"broken"}]}`))
		})
		types, err := c.PublicCloudVM().StorageType().List(ctx, nil)
		if err == nil {
			t.Fatalf("one broken item must fail the whole listing, got %d types", len(types))
		}
		if !strings.Contains(err.Error(), "st-2") {
			t.Fatalf("the error must name the offending item, got: %v", err)
		}
	})
}

// TestCapacitySubjectNamesAnUnidentifiedObject pins the diagnostic on the very
// payload these errors exist for: a malformed one, which may well carry no id.
func TestCapacitySubjectNamesAnUnidentifiedObject(t *testing.T) {
	var d PublicCloudVMDisk
	err := decodeInto(t, `{"position":1}`, &d)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "no id in the payload") {
		t.Fatalf("an id-less object must be described explicitly, got: %v", err)
	}
}
