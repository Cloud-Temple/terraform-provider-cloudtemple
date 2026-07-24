package client

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"
)

// This file is an OPT-IN, human-gated live probe for the template->image rename.
// With CT_IMAGE_LIVE_PROBE unset it Skips immediately, so a plain `go test ./...`
// touches nothing. It gathers the §5 LIVE evidence a spec cannot prove: that the
// renamed endpoint GET /vm_instances/v1/images is actually deployed on DEV, that
// the renamed client decodes its bare-array response, and (write phase) that a VM
// create accepts `imageId` and reads back `image{id,name}`.
//
// SAFETY: this probe FAIL-CLOSES unless the configured host is exactly
// imageProbeHost (the DEV broker), so a stale/prod CLOUDTEMPLE_HTTP_ADDR can never
// be hit by accident — /images does not exist on PROD anyway. The write phase is
// gated behind a SECOND flag; it creates exactly one stopped VM and deletes it,
// with a name-based cleanup (t.Cleanup runs even on a mid-probe Fatal) so a leak
// is always cleaned or logged loudly.
const (
	imageProbeHost   = "api.shiva.dev.ctlabs.me"
	imageProbeVMName = "tf-image-live-probe"
	imageProbeBogus  = "00000000-0000-0000-0000-000000000000"
)

func TestPublicCloudVMImageLiveProbe(t *testing.T) {
	if os.Getenv("CT_IMAGE_LIVE_PROBE") != "1" {
		t.Skip("opt-in live probe; set CT_IMAGE_LIVE_PROBE=1 (read-only) and CT_IMAGE_LIVE_PROBE_WRITE=1 (controlled VM create/delete)")
	}

	cfg := DefaultConfig()
	if cfg.Address != imageProbeHost {
		t.Fatalf("refusing to run: CLOUDTEMPLE_HTTP_ADDR=%q but this probe is authorised ONLY for %q (/images is DEV-only)", cfg.Address, imageProbeHost)
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	tok, err := c.Token(ctx)
	if err != nil {
		t.Fatalf("auth failed (check CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID): %v", err)
	}
	t.Logf("AUTH ok: host=%s tenant=%s user=%s", cfg.Address, tok.TenantID(), tok.UserID())

	rawGet := func(path string, args ...interface{}) (int, string) {
		r := c.newRequest("GET", path, args...)
		resp, err := c.doRequest(ctx, r)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer closeResponseBody(resp)
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	// ---- Phase 1: read-only proof that /images is live and decodes ----------
	imgs, err := c.PublicCloudVM().Image().List(ctx, nil)
	if err != nil {
		t.Fatalf("PHASE1 List(/vm_instances/v1/images) failed: %v", err)
	}
	if len(imgs) == 0 {
		t.Fatalf("PHASE1 /images returned an empty catalogue; expected at least one image on DEV")
	}
	first := imgs[0]
	t.Logf("PHASE1 /images returned %d images; first: id=%s name=%q osFamily=%q imageType=%q diskSizesGb=%v",
		len(imgs), first.ID, first.Name, first.OsFamily, first.ImageType, first.DiskSizesGb)

	byID, err := c.PublicCloudVM().Image().Read(ctx, first.ID)
	if err != nil || byID == nil || byID.ID != first.ID {
		t.Fatalf("PHASE1 Read(image %s) failed: img=%+v err=%v", first.ID, byID, err)
	}
	t.Logf("PHASE1 Read by id ok: %s (%q)", byID.ID, byID.Name)

	st, _ := rawGet("/vm_instances/v1/images/%s", imageProbeBogus)
	t.Logf("PHASE1 absent-signal: GET images/<bogus> -> HTTP %d (expect 404 => fail-closed absence contract)", st)

	if os.Getenv("CT_IMAGE_LIVE_PROBE_WRITE") != "1" {
		t.Log("PHASE2 skipped (set CT_IMAGE_LIVE_PROBE_WRITE=1 to run the controlled VM create/delete)")
		return
	}

	// ---- Phase 2: controlled write — create a VM from `imageId`, read back `image` ----
	waitOpts := &WaiterOptions{Logger: func(m string) { t.Logf("  activity: %s", m) }}

	// Resolve a compatible AZ + family from the raw image JSON, and a Private
	// Backbone network (vpc == nil) from the catalogue. All read-only.
	_, rawImg := rawGet("/vm_instances/v1/images/%s", first.ID)
	var compat struct {
		CompatibleFamilies          []string `json:"compatibleFamilies"`
		CompatibleAvailabilityZones []string `json:"compatibleAvailabilityZones"`
	}
	if err := json.Unmarshal([]byte(rawImg), &compat); err != nil {
		t.Fatalf("PHASE2 could not decode image compatibility from raw JSON: %v", err)
	}
	if len(compat.CompatibleFamilies) == 0 || len(compat.CompatibleAvailabilityZones) == 0 {
		t.Skipf("PHASE2 image %s has no compatible family/AZ advertised (families=%v azs=%v); cannot build a valid create — skipping write phase",
			first.ID, compat.CompatibleFamilies, compat.CompatibleAvailabilityZones)
	}
	familyID := compat.CompatibleFamilies[0]
	azID := compat.CompatibleAvailabilityZones[0]

	nets, err := c.PublicCloudVM().Network().List(ctx)
	if err != nil {
		t.Fatalf("PHASE2 Network().List failed: %v", err)
	}
	var pbNet string
	for _, n := range nets {
		if n != nil && n.VPC == nil { // Private Backbone (no vpc block)
			pbNet = n.ID
			break
		}
	}
	if pbNet == "" {
		t.Skipf("PHASE2 no Private Backbone network found in the catalogue; skipping write phase")
	}

	// The API requires a valid backupPolicyId on create (an empty string is a 400).
	policies, err := c.PublicCloudVM().BackupPolicy().List(ctx)
	if err != nil {
		t.Fatalf("PHASE2 BackupPolicy().List failed: %v", err)
	}
	var backupPolicyID string
	if len(policies) > 0 && policies[0] != nil {
		backupPolicyID = policies[0].ID
	}
	if backupPolicyID == "" {
		t.Skipf("PHASE2 no backup policy found in the catalogue; skipping write phase (create requires a valid backupPolicyId)")
	}
	t.Logf("PHASE2 create plan: image=%s az=%s family=%s pbNetwork=%s backupPolicy=%s name=%s", first.ID, azID, familyID, pbNet, backupPolicyID, imageProbeVMName)

	// Name-based cleanup guarantees no orphan even on a mid-probe Fatal.
	cleanupByName := func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer ccancel()
		leftovers, lerr := c.PublicCloudVM().Instance().List(cctx, &PublicCloudVMInstanceFilter{Name: imageProbeVMName})
		if lerr != nil {
			t.Logf("CLEANUP list by name failed: %v -- MANUAL CLEANUP MAY BE NEEDED for %q", lerr, imageProbeVMName)
			return
		}
		for _, vm := range leftovers {
			if vm == nil || vm.Name != imageProbeVMName {
				continue
			}
			loc, derr := c.PublicCloudVM().Instance().Delete(cctx, vm.ID)
			if derr != nil {
				t.Logf("CLEANUP delete %s failed: %v -- MANUAL CLEANUP MAY BE NEEDED", vm.ID, derr)
				continue
			}
			if _, werr := c.Activity().WaitForCompletion(cctx, loc, waitOpts); werr != nil {
				t.Logf("CLEANUP delete activity for %s did not complete: %v -- MANUAL CLEANUP MAY BE NEEDED", vm.ID, werr)
				continue
			}
			t.Logf("CLEANUP deleted leftover VM %s", vm.ID)
		}
	}
	t.Cleanup(cleanupByName)

	// Create with the RENAMED ImageID (json:"imageId") — the core write contract.
	// PowerState "off" keeps it light (no boot).
	createAct, err := c.PublicCloudVM().Instance().Create(ctx, &CreateVMInstanceRequest{
		Name:               imageProbeVMName,
		AvailabilityZoneID: azID,
		ImageID:            first.ID,
		InstanceFamilyID:   familyID,
		CPU:                2,
		Memory:             4,
		BackupPolicyID:     backupPolicyID,
		NetworkInterfaces:  []CreateVMInstanceNIC{{DeviceIndex: 0, NetworkID: pbNet}},
		PowerState:         "off",
	})
	if err != nil {
		t.Fatalf("PHASE2 Create (imageId=%s) failed: %v", first.ID, err)
	}
	if _, err := c.Activity().WaitForCompletion(ctx, createAct, waitOpts); err != nil {
		t.Fatalf("PHASE2 create activity failed: %v", err)
	}

	// Resolve the created VM by name and read it back: the read must expose the
	// image under `image{id,name}` (the renamed VM read contract) and match.
	created, err := c.PublicCloudVM().Instance().List(ctx, &PublicCloudVMInstanceFilter{Name: imageProbeVMName})
	if err != nil {
		t.Fatalf("PHASE2 list-by-name after create failed: %v", err)
	}
	var vmID string
	for _, vm := range created {
		if vm != nil && vm.Name == imageProbeVMName {
			vmID = vm.ID
			break
		}
	}
	if vmID == "" {
		t.Fatalf("PHASE2 created VM %q not found by name after create activity completed", imageProbeVMName)
	}
	vm, err := c.PublicCloudVM().Instance().Read(ctx, vmID)
	if err != nil || vm == nil {
		t.Fatalf("PHASE2 Read(vm %s) failed: vm=%+v err=%v", vmID, vm, err)
	}
	t.Logf("PHASE2 created VM %s: image={id:%s name:%q} instanceFamily={id:%s}", vm.ID, vm.Image.ID, vm.Image.Name, vm.InstanceFamily.ID)
	if vm.Image.ID != first.ID {
		t.Fatalf("PHASE2 read-back image.id=%q, want the create imageId %q (the image read contract is broken)", vm.Image.ID, first.ID)
	}

	// Delete and confirm absence (fail-closed 404 contract, live).
	delAct, err := c.PublicCloudVM().Instance().Delete(ctx, vmID)
	if err != nil {
		t.Fatalf("PHASE2 Delete(vm %s) failed: %v", vmID, err)
	}
	if _, err := c.Activity().WaitForCompletion(ctx, delAct, waitOpts); err != nil {
		t.Fatalf("PHASE2 delete activity failed: %v", err)
	}
	gone, err := c.PublicCloudVM().Instance().Read(ctx, vmID)
	if err != nil {
		t.Fatalf("PHASE2 post-delete Read errored (want (nil,nil) for 404): %v", err)
	}
	if gone != nil {
		t.Fatalf("PHASE2 VM %s still present after delete: %+v", vmID, gone)
	}
	t.Logf("PHASE2 VM %s deleted and confirmed absent (0-orphan)", vmID)
	t.Log("PROBE DONE")
}
