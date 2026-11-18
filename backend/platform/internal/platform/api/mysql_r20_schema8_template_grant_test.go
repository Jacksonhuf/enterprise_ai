package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
)

func activeGrantMatches(st app.State, subjectType, subjectID, resourceType, resourceID, action string) int {
	n := 0
	for _, g := range st.Grants {
		if g.Active && g.SubjectType == subjectType && g.SubjectID == subjectID && g.ResourceType == resourceType && g.ResourceID == resourceID && g.Action == action {
			n++
		}
	}
	return n
}

// TestR20Schema8DualHostTemplateCopyRejectsStaleRevision rejects template copy when
// the template draft moved on schema8 shared MySQL.
func TestR20Schema8DualHostTemplateCopyRejectsStaleRevision(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	source := f.store.Read().Apps["channel-replenishment"]
	stale := source.Revision
	cfg := source.Draft
	cfg.Basic.Name = "schema8-template-moved"
	body := struct {
		Expected string `json:"expected_draft_revision"`
		app.Config
	}{stale, cfg}
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", f.host.URL+"/api/v1/cognitive/apps/"+source.ID, bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer demo-engineer")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal("template draft save", resp.StatusCode)
	}
	fresh := f.store.Read().Apps[source.ID].Revision
	if fresh == stale {
		t.Fatal("revision not bumped")
	}
	appCount := len(f.store.Read().Apps)
	copyBody, _ := json.Marshal(map[string]any{
		"creation_mode": "TEMPLATE", "name": "schema8-stale-copy", "template_id": source.ID, "expected_template_revision": stale,
	})
	for _, address := range []string{f.host.URL, host.URL} {
		req, _ := http.NewRequest("POST", address+"/api/v1/cognitive/apps", bytes.NewReader(copyBody))
		req.Header.Set("Authorization", "Bearer demo-engineer")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 409 {
			t.Fatal("stale template copy", address, resp.StatusCode)
		}
	}
	if len(f.store.Read().Apps) != appCount {
		t.Fatal("stale copy mutated app catalog")
	}
	decode(t, f.request("demo-engineer", "POST", "/api/v1/cognitive/apps", map[string]any{
		"creation_mode": "TEMPLATE", "name": "schema8-fresh-copy", "template_id": source.ID, "expected_template_revision": fresh,
	}, 200))
}

// TestR20Schema8DualHostGrantRevokePostRace interleaves grant POST and DELETE on dual HTTP hosts.
func TestR20Schema8DualHostGrantRevokePostRace(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	grantID := "grant-app"
	if activeGrantMatches(f.store.Read(), "GROUP", "all-users", "APPLICATION", "channel-replenishment", "application.use") != 1 {
		t.Fatal("fixture grant missing")
	}
	grantsBefore := len(f.store.Read().Grants)
	grantAudits := countGrantAudit(f.store.Read(), "authorization.grant")
	revokeAudits := countGrantAudit(f.store.Read(), "authorization.revoke")
	body, _ := json.Marshal(grantPayload("GROUP", "all-users", "APPLICATION", "channel-replenishment", "application.use"))
	type outcome struct {
		op     string
		status int
	}
	var wg sync.WaitGroup
	out := make(chan outcome, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		req, _ := http.NewRequest("DELETE", f.host.URL+grantPath+"/"+grantID, nil)
		req.Header.Set("Authorization", "Bearer demo-admin")
		resp, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Error(e)
			out <- outcome{"delete", 0}
			return
		}
		resp.Body.Close()
		out <- outcome{"delete", resp.StatusCode}
	}()
	go func() {
		defer wg.Done()
		req, _ := http.NewRequest("POST", host.URL+grantPath, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer demo-admin")
		req.Header.Set("Content-Type", "application/json")
		resp, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Error(e)
			out <- outcome{"post", 0}
			return
		}
		resp.Body.Close()
		out <- outcome{"post", resp.StatusCode}
	}()
	wg.Wait()
	close(out)
	for o := range out {
		if o.status != 200 {
			t.Fatal(o.op, o.status)
		}
	}
	st := f.store.Read()
	if active := activeGrantMatches(st, "GROUP", "all-users", "APPLICATION", "channel-replenishment", "application.use"); active > 1 {
		t.Fatal("duplicate active grants", active)
	}
	if len(st.Grants) > grantsBefore+1 {
		t.Fatal("unexpected grant rows", grantsBefore, len(st.Grants))
	}
	grantDelta := countGrantAudit(st, "authorization.grant") - grantAudits
	revokeDelta := countGrantAudit(st, "authorization.revoke") - revokeAudits
	if grantDelta < 0 || revokeDelta < 0 || grantDelta > 1 || revokeDelta > 1 {
		t.Fatal("audit deltas", grantDelta, revokeDelta)
	}
	if grantDelta+revokeDelta < 1 {
		t.Fatal("no authorization mutation recorded", grantDelta, revokeDelta)
	}
}

// TestR20Schema8DualHostTemplateCopyConcurrentFresh allows two distinct copies when
// the template revision is current on both hosts (not a CAS conflict).
func TestR20Schema8DualHostTemplateCopyConcurrentFresh(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	source := f.store.Read().Apps["channel-replenishment"]
	rev := source.Revision
	before := len(f.store.Read().Apps)
	type result struct {
		status int
		id     string
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i, address := range []string{f.host.URL, host.URL} {
		wg.Add(1)
		go func(i int, address string) {
			defer wg.Done()
			payload, _ := json.Marshal(map[string]any{
				"creation_mode": "TEMPLATE",
				"name":          fmt.Sprintf("schema8-dual-copy-%d", i),
				"template_id":   source.ID,
				"expected_template_revision": rev,
			})
			req, _ := http.NewRequest("POST", address+"/api/v1/cognitive/apps", bytes.NewReader(payload))
			req.Header.Set("Authorization", "Bearer demo-engineer")
			req.Header.Set("Content-Type", "application/json")
			resp, e := http.DefaultClient.Do(req)
			if e != nil {
				t.Error(e)
				results <- result{0, ""}
				return
			}
			defer resp.Body.Close()
			var created app.App
			json.NewDecoder(resp.Body).Decode(&created)
			results <- result{resp.StatusCode, created.ID}
		}(i, address)
	}
	wg.Wait()
	close(results)
	ids := map[string]bool{}
	for r := range results {
		if r.status != 200 || r.id == "" || ids[r.id] {
			t.Fatal("copy failed", r)
		}
		ids[r.id] = true
	}
	if len(f.store.Read().Apps) != before+2 {
		t.Fatal("expected two new apps", before, len(f.store.Read().Apps))
	}
}