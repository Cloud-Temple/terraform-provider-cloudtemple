package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

// TestPublicCloudVMInstanceListDecode pins the decode of the wrapped
// ({"vms":[...],"total":N}), camelCase GET /vm_instances/v1/virtual_machines
// response: nested {id,name}
// refs (az, image, instanceFamily), int fields (vcpu, ramGb, disksSizeGb),
// the bool guestToolsInstalled, and a nullable backupPolicy that decodes to nil.
func TestPublicCloudVMInstanceListDecode(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/virtual_machines" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total":2,"vms":[
		  {"id":"vm-1","name":"web","status":"running",
		   "az":{"id":"az-1","name":"fr1-az01"},
		   "image":{"id":"img-1","name":"rocky-9"},
		   "instanceFamily":{"id":"fam-1","name":"general"},
		   "vcpu":2,"ramGb":4,"disksSizeGb":40,
		   "backupPolicy":{"id":"bp-1","name":"daily"},
		   "guestToolsInstalled":true,
		   "createdAt":"2026-04-14T12:50:41.881814","updatedAt":"2026-04-14T12:50:41.881814"},
		  {"id":"vm-2","name":"db","status":"stopped",
		   "az":{"id":"az-1","name":"fr1-az01"},
		   "image":{"id":"img-1","name":"rocky-9"},
		   "instanceFamily":{"id":"fam-1","name":"general"},
		   "vcpu":1,"ramGb":2,"disksSizeGb":20,
		   "backupPolicy":null,
		   "guestToolsInstalled":false,
		   "createdAt":"2026-04-14T12:50:41.881814","updatedAt":"2026-04-14T12:50:41.881814"}
		]}`))
	})

	vms, err := c.PublicCloudVM().Instance().List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(vms) != 2 {
		t.Fatalf("want 2 vms, got %d", len(vms))
	}
	first := vms[0]
	if first.ID != "vm-1" || first.Name != "web" || first.Status != "running" {
		t.Fatalf("scalar fields not decoded: %+v", first)
	}
	if first.AZ.ID != "az-1" || first.AZ.Name != "fr1-az01" {
		t.Fatalf("az ref not decoded: %+v", first.AZ)
	}
	if first.Image.ID != "img-1" || first.InstanceFamily.ID != "fam-1" {
		t.Fatalf("image/family refs not decoded: %+v", first)
	}
	if first.VCPU != 2 || first.RAMGb != 4 || first.DisksSizeGb != 40 {
		t.Fatalf("int fields not decoded: vcpu=%d ramGb=%d disksSizeGb=%d", first.VCPU, first.RAMGb, first.DisksSizeGb)
	}
	if !first.GuestToolsInstalled {
		t.Fatalf("guestToolsInstalled should decode to true")
	}
	if first.BackupPolicy == nil || first.BackupPolicy.ID != "bp-1" {
		t.Fatalf("backupPolicy should decode to a non-nil ref: %+v", first.BackupPolicy)
	}
	if vms[1].BackupPolicy != nil {
		t.Fatalf("a null backupPolicy must decode to nil, got %+v", vms[1].BackupPolicy)
	}
}

// TestPublicCloudVMInstanceListPaginates is the E0-6 mutation-killer: the client
// MUST page through the full result set (limit+offset), not return only the
// first page nor loop forever. The stub serves a dataset of pageSize+extra items;
// a correct client fetches page 0 (full) and page 1 (short) and stops.
func TestPublicCloudVMInstanceListPaginates(t *testing.T) {
	ctx := context.Background()
	const total = publicCloudVMInstanceListPageSize + 3 // 203: two pages, second short

	var offsetsSeen []int
	var limitsSeen []int
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		offsetsSeen = append(offsetsSeen, offset)
		limitsSeen = append(limitsSeen, limit)

		page := []map[string]any{}
		for n := offset; n < offset+limit && n < total; n++ {
			page = append(page, map[string]any{"id": fmt.Sprintf("vm-%d", n), "name": fmt.Sprintf("vm%d", n)})
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"vms": page, "total": len(page)})
	})

	vms, err := c.PublicCloudVM().Instance().List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(vms) != total {
		t.Fatalf("pagination lost items: want %d, got %d (a non-paginating client would return %d)", total, len(vms), publicCloudVMInstanceListPageSize)
	}
	// Order is preserved across pages and every item is distinct.
	if vms[0].ID != "vm-0" || vms[total-1].ID != fmt.Sprintf("vm-%d", total-1) {
		t.Fatalf("page concatenation order wrong: first=%s last=%s", vms[0].ID, vms[total-1].ID)
	}
	if len(offsetsSeen) != 2 || offsetsSeen[0] != 0 || offsetsSeen[1] != publicCloudVMInstanceListPageSize {
		t.Fatalf("offsets seen = %v, want [0 %d]", offsetsSeen, publicCloudVMInstanceListPageSize)
	}
	for _, l := range limitsSeen {
		if l != publicCloudVMInstanceListPageSize {
			t.Fatalf("every page must request limit=%d, got %v", publicCloudVMInstanceListPageSize, limitsSeen)
		}
	}
}

// TestPublicCloudVMInstanceListRefusesRunawayPagination is the E0-6 fail-closed
// guard: if the server ignores `offset` and keeps returning the same full page,
// the client must ERROR after a bounded number of requests — never loop forever
// (unbounded memory / spin) and never silently truncate. Deleting the progress
// guard turns this RED with an infinite loop.
func TestPublicCloudVMInstanceListRefusesRunawayPagination(t *testing.T) {
	ctx := context.Background()
	calls := 0
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		// Ignore offset entirely: always return the SAME full page of distinct ids.
		page := make([]map[string]any, publicCloudVMInstanceListPageSize)
		for n := 0; n < publicCloudVMInstanceListPageSize; n++ {
			page[n] = map[string]any{"id": fmt.Sprintf("vm-%d", n)}
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"vms": page, "total": len(page)})
	})

	_, err := c.PublicCloudVM().Instance().List(ctx, nil)
	if err == nil {
		t.Fatal("a server that ignores offset (repeats a full page) must fail closed, not loop or truncate")
	}
	if calls != 2 {
		t.Fatalf("runaway must be caught after exactly 2 pages (first full page of new ids, second all-duplicate), got %d calls", calls)
	}
}

// TestPublicCloudVMInstanceListStrictLaterPagePartial pins that the strict
// 200-only contract holds across EVERY page: a 206 on a later page (after a full
// 200-OK first page) must fail closed, not return the partial-but-first page as a
// complete listing (which E0-9 could misread as proof of absence).
func TestPublicCloudVMInstanceListStrictLaterPagePartial(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if offset == 0 {
			page := make([]map[string]any, publicCloudVMInstanceListPageSize)
			for n := 0; n < publicCloudVMInstanceListPageSize; n++ {
				page[n] = map[string]any{"id": fmt.Sprintf("vm-%d", n)}
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"vms": page, "total": len(page)})
			return
		}
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte(`[{"id":"vm-200"}]`))
	})
	if _, err := c.PublicCloudVM().Instance().ListStrict(ctx, nil); err == nil {
		t.Fatal("a 206 on a later page must fail closed (strict 200-only across all pages)")
	}
}

// TestPublicCloudVMInstanceListFilters pins the mapping of the filter struct to
// query parameters (including familyId, whose query name differs from the create
// body's instanceFamilyId).
func TestPublicCloudVMInstanceListFilters(t *testing.T) {
	ctx := context.Background()
	var q url.Values
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"vms":[],"total":0}`))
	})

	_, err := c.PublicCloudVM().Instance().List(ctx, &PublicCloudVMInstanceFilter{
		Name:               "web",
		Status:             "running",
		AvailabilityZoneID: "az-1",
		FamilyID:           "fam-1",
		OrderBy:            "name",
		OrderDir:           "asc",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := map[string]string{
		"name":               "web",
		"status":             "running",
		"availabilityZoneId": "az-1",
		"familyId":           "fam-1",
		"orderBy":            "name",
		"orderDir":           "asc",
	}
	for k, v := range want {
		if got := q.Get(k); got != v {
			t.Fatalf("query %q = %q, want %q (full query: %v)", k, got, v, q)
		}
	}
}

// TestPublicCloudVMInstanceListStrict pins the E0-9 evidence channel: only a
// complete HTTP 200 is a usable listing. A 206 is partial and MUST fail closed
// (never accepted as a complete listing that could prove an absence); a 403 is
// not evidence either.
func TestPublicCloudVMInstanceListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 returns the listing", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"vms":[{"id":"vm-1","name":"web"}],"total":1}`))
		})
		vms, err := c.PublicCloudVM().Instance().ListStrict(ctx, nil)
		if err != nil {
			t.Fatalf("ListStrict 200: %v", err)
		}
		if len(vms) != 1 || vms[0].ID != "vm-1" {
			t.Fatalf("unexpected listing: %+v", vms)
		}
	})

	t.Run("206 partial fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte(`{"vms":[{"id":"vm-1","name":"web"}],"total":1}`))
		})
		if _, err := c.PublicCloudVM().Instance().ListStrict(ctx, nil); err == nil {
			t.Fatal("a 206 partial listing must fail closed (it cannot prove an absence)")
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		if _, err := c.PublicCloudVM().Instance().ListStrict(ctx, nil); err == nil {
			t.Fatal("a 403 listing must fail closed")
		}
	})
}

// TestPublicCloudVMInstanceRead pins the state-safety contract of Read: a
// positive 404 is absence (nil,nil); a 403 (or any other non-OK) fails closed
// with an error and never maps to absence.
func TestPublicCloudVMInstanceRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 by id decodes", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"vm-1","name":"web","status":"running","vcpu":2,"ramGb":4}`))
		})
		vm, err := c.PublicCloudVM().Instance().Read(ctx, "vm-1")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if vm == nil || vm.ID != "vm-1" || vm.VCPU != 2 {
			t.Fatalf("bad vm: %+v", vm)
		}
	})

	t.Run("404 is absence (nil,nil)", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		vm, err := c.PublicCloudVM().Instance().Read(ctx, "missing")
		if err != nil {
			t.Fatalf("404 must not error, got %v", err)
		}
		if vm != nil {
			t.Fatalf("404 must return a nil vm, got %+v", vm)
		}
	})

	t.Run("403 fails closed (error, not absence)", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		vm, err := c.PublicCloudVM().Instance().Read(ctx, "denied")
		if err == nil {
			t.Fatalf("403 must fail closed with an error, got vm %+v", vm)
		}
		if vm != nil {
			t.Fatalf("403 must not return a vm")
		}
	})
}

// TestPublicCloudVMInstanceCreate pins the create body encoding and the async
// contract: camelCase keys, powerState passed through, disks[] deliberately
// absent (the resource never creates disks), and the activityId returned from
// the Location header.
func TestPublicCloudVMInstanceCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("encodes camelCase body, omits disks, returns activityId", func(t *testing.T) {
		var body map[string]any
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/vm_instances/v1/virtual_machines" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("body not JSON: %v (%s)", err, raw)
			}
			w.Header().Set("Location", "act-create-1")
			w.WriteHeader(http.StatusCreated)
		})

		activityID, err := c.PublicCloudVM().Instance().Create(ctx, &CreateVMInstanceRequest{
			Name:               "web",
			AvailabilityZoneID: "az-1",
			ImageID:            "img-1",
			InstanceFamilyID:   "fam-1",
			CPU:                2,
			Memory:             4,
			BackupPolicyID:     "bp-1",
			NetworkInterfaces:  []CreateVMInstanceNIC{{DeviceIndex: 0, NetworkID: "net-1"}},
			PowerState:         "on",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if activityID != "act-create-1" {
			t.Fatalf("Create must return the Location activityId, got %q", activityID)
		}
		for _, k := range []string{"name", "availabilityZoneId", "imageId", "instanceFamilyId", "cpu", "memory", "backupPolicyId", "networkInterfaces", "powerState"} {
			if _, ok := body[k]; !ok {
				t.Fatalf("create body missing camelCase key %q (body: %v)", k, body)
			}
		}
		// Negative assertion: the old key must be gone, so a revert to templateId turns this RED.
		if _, ok := body["templateId"]; ok {
			t.Fatalf("create body must NOT contain the old key templateId (body: %v)", body)
		}
		if _, ok := body["disks"]; ok {
			t.Fatalf("create body must NOT contain disks[]: the resource never creates disks (body: %v)", body)
		}
		nics, ok := body["networkInterfaces"].([]any)
		if !ok || len(nics) != 1 {
			t.Fatalf("networkInterfaces not encoded as a 1-element array: %v", body["networkInterfaces"])
		}
		nic := nics[0].(map[string]any)
		if _, ok := nic["deviceIndex"]; !ok {
			t.Fatalf("nic must encode deviceIndex (camelCase): %v", nic)
		}
		if nic["networkId"] != "net-1" {
			t.Fatalf("nic networkId = %v, want net-1", nic["networkId"])
		}
		// ip_address unset and cloud_init nil must be omitted.
		if _, ok := nic["ipAddress"]; ok {
			t.Fatalf("an unset ipAddress must be omitted, got %v", nic["ipAddress"])
		}
		if _, ok := body["cloudInit"]; ok {
			t.Fatalf("a nil cloudInit must be omitted, got %v", body["cloudInit"])
		}
	})

	t.Run("empty Location fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			// 201 without a Location header: the write was accepted but we cannot
			// track it -> must error, never return ("", nil).
			w.WriteHeader(http.StatusCreated)
		})
		if _, err := c.PublicCloudVM().Instance().Create(ctx, &CreateVMInstanceRequest{Name: "web"}); err == nil {
			t.Fatal("a 201 without a Location must fail (untrackable write)")
		}
	})
}

// TestPublicCloudVMInstanceDiffDrivenBodies pins that PATCH and resize send ONLY
// the changed dimensions (pointer + omitempty), so an update never clobbers an
// attribute the user did not touch.
func TestPublicCloudVMInstanceDiffDrivenBodies(t *testing.T) {
	ctx := context.Background()

	t.Run("PatchMetadata with only name omits backupPolicyId", func(t *testing.T) {
		var body map[string]any
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Location", "act-patch-1")
			w.WriteHeader(http.StatusCreated)
		})
		name := "web2"
		activityID, err := c.PublicCloudVM().Instance().PatchMetadata(ctx, "vm-1", &PatchVMInstanceRequest{Name: &name})
		if err != nil {
			t.Fatalf("PatchMetadata: %v", err)
		}
		if activityID != "act-patch-1" {
			t.Fatalf("want activityId act-patch-1, got %q", activityID)
		}
		if body["name"] != "web2" {
			t.Fatalf("name not sent: %v", body)
		}
		if _, ok := body["backupPolicyId"]; ok {
			t.Fatalf("an unchanged backupPolicyId must be omitted, got %v", body["backupPolicyId"])
		}
	})

	t.Run("Resize with only cpu omits memory", func(t *testing.T) {
		var body map[string]any
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/resize" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Location", "act-resize-1")
			w.WriteHeader(http.StatusCreated)
		})
		cpu := 4
		activityID, err := c.PublicCloudVM().Instance().Resize(ctx, "vm-1", &ResizeVMInstanceRequest{CPU: &cpu})
		if err != nil {
			t.Fatalf("Resize: %v", err)
		}
		if activityID != "act-resize-1" {
			t.Fatalf("want activityId act-resize-1, got %q", activityID)
		}
		if body["cpu"] != float64(4) {
			t.Fatalf("cpu not sent: %v", body)
		}
		if _, ok := body["memory"]; ok {
			t.Fatalf("an unchanged memory must be omitted, got %v", body["memory"])
		}
	})
}

// TestPublicCloudVMInstancePowerAndDelete pins the method+path of the bodyless
// power transitions and the delete, and that each returns the Location activityId.
func TestPublicCloudVMInstancePowerAndDelete(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name       string
		call       func(c *Client) (string, error)
		wantMethod string
		wantPath   string
		activityID string
	}{
		{"start", func(c *Client) (string, error) { return c.PublicCloudVM().Instance().Start(ctx, "vm-1") }, http.MethodPost, "/vm_instances/v1/virtual_machines/vm-1/start", "act-start"},
		{"stop", func(c *Client) (string, error) { return c.PublicCloudVM().Instance().Stop(ctx, "vm-1") }, http.MethodPost, "/vm_instances/v1/virtual_machines/vm-1/stop", "act-stop"},
		{"delete", func(c *Client) (string, error) { return c.PublicCloudVM().Instance().Delete(ctx, "vm-1") }, http.MethodDelete, "/vm_instances/v1/virtual_machines/vm-1", "act-delete"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.wantMethod || r.URL.Path != tc.wantPath {
					t.Errorf("unexpected request: %s %s (want %s %s)", r.Method, r.URL.Path, tc.wantMethod, tc.wantPath)
				}
				w.Header().Set("Location", tc.activityID)
				w.WriteHeader(http.StatusCreated)
			})
			got, err := tc.call(c)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if got != tc.activityID {
				t.Fatalf("%s activityId = %q, want %q", tc.name, got, tc.activityID)
			}
		})
	}
}
