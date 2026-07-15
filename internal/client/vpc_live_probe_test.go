package client

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

// This file is an OPT-IN, human-gated live probe. It is NEVER part of the
// hermetic `all` cycle: with CT_VPC_LIVE_PROBE unset it Skips immediately, so a
// plain `go test ./...` touches nothing. It exists to gather the §5 LIVE evidence
// that the swagger alone cannot prove (a spec is never proof of deployed
// behaviour), feeding the C0 contract-drift gate (R-Q9) and clearing the
// async-create premise (R-Q2/R-Q10).
//
// SAFETY: Paul GO'd a controlled write probe against the dev environment
// api.shiva.dev.ctlabs.me, on private network fsn-pn-02, creating then deleting
// one static IP and one floating IP of test. The probe FAIL-CLOSES unless the
// configured host is exactly probeHost, so a stale/prod CLOUDTEMPLE_HTTP_ADDR can
// never be hit by accident. Everything created is deleted (t.Cleanup runs even on
// a mid-probe Fatal); created ids are logged loudly so any leak can be cleaned by
// hand.
const (
	probeHost      = "api.shiva.dev.ctlabs.me"
	probePN        = "7f5d6a7a-fbe6-45f4-9a51-c1ea991a5d97" // fsn-pn-02
	probeBogusUUID = "00000000-0000-0000-0000-000000000000"
)

func TestVPCLiveContractProbe(t *testing.T) {
	if os.Getenv("CT_VPC_LIVE_PROBE") != "1" {
		t.Skip("opt-in live probe; set CT_VPC_LIVE_PROBE=1 (read-only) and CT_VPC_LIVE_PROBE_WRITE=1 (controlled write)")
	}

	cfg := DefaultConfig()
	if cfg.Address != probeHost {
		t.Fatalf("refusing to run: CLOUDTEMPLE_HTTP_ADDR=%q but this probe is authorised ONLY for %q", cfg.Address, probeHost)
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	// Auth + identity. Log the tenant so the human can confirm the target; if a
	// recette tenant allowlist is provided, assert it (extra fail-closed guard).
	tok, err := c.Token(ctx)
	if err != nil {
		t.Fatalf("auth failed (check CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID): %v", err)
	}
	t.Logf("AUTH ok: host=%s tenant=%s user=%s", cfg.Address, tok.TenantID(), tok.UserID())
	if want := os.Getenv("CLOUDTEMPLE_RECETTE_TENANT_ID"); want != "" && tok.TenantID() != want {
		t.Fatalf("refusing to run: authenticated tenant %q != allowlisted CLOUDTEMPLE_RECETTE_TENANT_ID %q", tok.TenantID(), want)
	}

	// rawGet captures the RAW status of a GET (the typed clients hide it behind
	// requireNotFoundOrOK, which is exactly the 403-vs-404 distinction we probe).
	rawGet := func(path string, args ...interface{}) (int, string) {
		r := c.newRequest("GET", path, args...)
		resp, err := c.doRequest(ctx, r)
		if err != nil {
			t.Fatalf("GET %s: %v", fmt.Sprintf(path, args...), err)
		}
		defer closeResponseBody(resp)
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	// ---- Phase 1: read-only evidence -------------------------------------
	st, _ := rawGet("/vpc/v1/static_ips/%s", probeBogusUUID)
	t.Logf("PHASE1 absent-signal: GET static_ips/<bogus> -> HTTP %d  (403=>indistinct access-denied, 404=>distinguishable absent)", st)

	stf, _ := rawGet("/vpc/v1/floating_ips/%s", probeBogusUUID)
	t.Logf("PHASE1 absent-signal: GET floating_ips/<bogus> -> HTTP %d", stf)

	stl, body := rawGet("/vpc/v1/private_networks/%s/static_ips", probePN)
	t.Logf("PHASE1 list pn static_ips -> HTTP %d", stl)
	if stl == 200 {
		var list []*StaticIP
		if err := json.Unmarshal([]byte(body), &list); err != nil {
			t.Logf("PHASE1 list decode error: %v (body head=%.120q)", err, body)
		} else {
			sources := map[string]int{}
			for _, si := range list {
				if si != nil {
					sources[si.Source]++
				}
			}
			t.Logf("PHASE1 pn has %d static IPs, live source distribution=%v", len(list), sources)
		}
	}

	if os.Getenv("CT_VPC_LIVE_PROBE_WRITE") != "1" {
		t.Log("PHASE2 skipped (set CT_VPC_LIVE_PROBE_WRITE=1 to run the controlled write probe)")
		return
	}

	// ---- Phase 2: controlled write ---------------------------------------
	waitOpts := &WaiterOptions{Logger: func(m string) { t.Logf("  activity: %s", m) }}

	var createdStaticID, createdFIPID string
	// deleteAndWait uses an INDEPENDENT context so a timeout on the main probe
	// never blocks the deletes that prevent orphans.
	deleteAndWait := func(kind, path, id string) {
		if id == "" {
			return
		}
		cctx, ccancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer ccancel()
		r := c.newRequest("DELETE", path, id)
		loc, err := c.doRequestAndReturnActivity(cctx, r)
		if err != nil {
			t.Logf("CLEANUP %s %s: DELETE failed: %v  -- MANUAL CLEANUP MAY BE NEEDED", kind, id, err)
			return
		}
		if _, err := c.Activity().WaitForCompletion(cctx, loc, waitOpts); err != nil {
			t.Logf("CLEANUP %s %s: delete activity did not complete: %v  -- MANUAL CLEANUP MAY BE NEEDED", kind, id, err)
			return
		}
		t.Logf("CLEANUP %s %s: deleted (activity %s)", kind, id, loc)
	}
	// Belt-and-suspenders: guarantee deletion even on a mid-probe Fatal.
	t.Cleanup(func() { deleteAndWait("static_ip", "/vpc/v1/static_ips/%s", createdStaticID) })
	t.Cleanup(func() { deleteAndWait("floating_ip", "/vpc/v1/floating_ips/%s", createdFIPID) })

	// (a) static IP create -- ASYNC (201 + Location) vs SYNC (201 + body)?
	mac := probeRandomLocalMAC()
	t.Logf("PHASE2 creating static IP on pn=%s mac=%s", probePN, mac)
	{
		r := c.newRequest("POST", "/vpc/v1/private_networks/%s/static_ips", probePN)
		r.obj = &CreateStaticIPRequest{MacAddress: mac, ResourceDescription: "vpc-live-probe"}
		resp, err := c.doRequest(ctx, r)
		if err != nil {
			t.Fatalf("POST static_ip transport error: %v", err)
		}
		status := resp.StatusCode
		loc := resp.Header.Get("Location")
		b, _ := io.ReadAll(resp.Body)
		closeResponseBody(resp)
		t.Logf("PHASE2 RESULT static_ip CREATE -> HTTP %d  Location=%q  body=%.300q", status, loc, string(b))

		switch {
		case loc != "":
			t.Log("PHASE2 => static_ip CREATE is ASYNC (Location activity present)")
			act, werr := c.Activity().WaitForCompletion(ctx, loc, waitOpts)
			if werr != nil {
				t.Fatalf("static_ip create activity failed: %v", werr)
			}
			createdStaticID = probeSingleActivityResult(t, act)
			t.Logf("PHASE2 static_ip activity Result UUID=%q", createdStaticID)
			if createdStaticID == "" {
				// Fail closed like the production WaitCreate (R-M1): an empty Result is
				// never a resolved id. But the IP WAS created live, so resolve its id by
				// MAC and stash it so t.Cleanup deletes it — a probe must never orphan.
				if si, rerr := c.VPC().StaticIP().ReadByMAC(ctx, mac); rerr == nil && si != nil && si.ID != "" {
					createdStaticID = si.ID
					t.Logf("PHASE2 empty Result; resolved the created static_ip id by MAC for cleanup: %s", si.ID)
				}
				t.Fatalf("static_ip create activity completed with an EMPTY Result (contradicts WaitCreate fail-closed); mac=%s -- CHECK MANUALLY if no id was resolved above", mac)
			}
		default:
			var sb struct {
				StaticIPID string `json:"static_ip_id"`
			}
			if json.Unmarshal(b, &sb) == nil && sb.StaticIPID != "" {
				createdStaticID = sb.StaticIPID
				t.Logf("PHASE2 => static_ip CREATE is SYNC (body static_ip_id=%q)", createdStaticID)
			} else {
				t.Fatalf("static_ip CREATE: neither Location nor body static_ip_id (HTTP %d, body %.300q)", status, string(b))
			}
		}
	}
	if createdStaticID != "" {
		stv, vb := rawGet("/vpc/v1/static_ips/%s", createdStaticID)
		t.Logf("PHASE2 verify static_ip %s -> HTTP %d", createdStaticID, stv)
		if stv == 200 {
			var si StaticIP
			if json.Unmarshal([]byte(vb), &si) == nil {
				t.Logf("PHASE2 created static_ip source=%q ip=%q mac=%q", si.Source, si.IPAddress, si.MacAddress)
			}
		}
		deleteAndWait("static_ip", "/vpc/v1/static_ips/%s", createdStaticID)
		sta, _ := rawGet("/vpc/v1/static_ips/%s", createdStaticID)
		t.Logf("PHASE2 post-delete absent-signal: GET static_ips/%s -> HTTP %d", createdStaticID, sta)
		createdStaticID = "" // already deleted; stop the cleanup re-delete
	}

	// (b) floating IP provision -- count=1, async, Result multiplicity (R-Q3)
	t.Log("PHASE2 provisioning floating IP {count:1}")
	{
		r := c.newRequest("POST", "/vpc/v1/floating_ips")
		r.obj = map[string]any{"count": 1}
		resp, err := c.doRequest(ctx, r)
		if err != nil {
			t.Fatalf("POST floating_ips transport error: %v", err)
		}
		status := resp.StatusCode
		loc := resp.Header.Get("Location")
		b, _ := io.ReadAll(resp.Body)
		closeResponseBody(resp)
		t.Logf("PHASE2 RESULT floating_ip CREATE -> HTTP %d  Location=%q  body=%.300q", status, loc, string(b))
		if loc == "" {
			t.Fatalf("floating_ip CREATE returned no Location (expected async); HTTP %d body=%.300q", status, string(b))
		}
		act, werr := c.Activity().WaitForCompletion(ctx, loc, waitOpts)
		if werr != nil {
			t.Fatalf("floating_ip create activity failed: %v", werr)
		}
		for name, stt := range act.State {
			t.Logf("PHASE2 floating_ip activity state=%q Result=%q", name, stt.Result)
			createdFIPID = stt.Result
		}
		t.Logf("PHASE2 floating_ip Result=%q (a comma in Result => multiplicity to handle)", createdFIPID)
		if createdFIPID == "" {
			// Fail loud rather than silently skip cleanup: a successful async create
			// whose Result we cannot read may have provisioned a floating IP we now
			// cannot address. There is no deterministic key to resolve it by (unlike a
			// static IP's MAC), so the human must check for an orphan.
			t.Fatalf("floating_ip create activity completed with an EMPTY Result; cannot resolve the created id for cleanup -- CHECK MANUALLY for an orphaned floating IP")
		}
	}
	if createdFIPID != "" {
		stv, vb := rawGet("/vpc/v1/floating_ips/%s", createdFIPID)
		t.Logf("PHASE2 verify floating_ip %s -> HTTP %d body=%.200q", createdFIPID, stv, vb)
		deleteAndWait("floating_ip", "/vpc/v1/floating_ips/%s", createdFIPID)
		fa, _ := rawGet("/vpc/v1/floating_ips/%s", createdFIPID)
		t.Logf("PHASE2 post-delete absent-signal: GET floating_ips/%s -> HTTP %d", createdFIPID, fa)
		createdFIPID = ""
	}

	t.Log("PROBE DONE")
}

func probeSingleActivityResult(t *testing.T, act *Activity) string {
	t.Helper()
	if act == nil || len(act.State) != 1 {
		t.Fatalf("expected exactly one activity state, got %#v", act)
	}
	for _, s := range act.State {
		return s.Result
	}
	return ""
}

func probeRandomLocalMAC() string {
	b := make([]byte, 6)
	_, _ = crand.Read(b)
	b[0] = (b[0] | 0x02) & 0xfe // locally administered, unicast
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3], b[4], b[5])
}
