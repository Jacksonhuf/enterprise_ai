package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
)

func publishAuditCount(st app.State) int {
	n := 0
	for _, e := range st.Audit {
		if e.Action == "application.publish" {
			n++
		}
	}
	return n
}

// TestR20Schema8DualHostReleaseCheckRejectsStaleRevision ensures release-checks
// honors draft revision after a concurrent save on schema8.
func TestR20Schema8DualHostReleaseCheckRejectsStaleRevision(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	before := f.store.Read().Apps["channel-replenishment"]
	stale := before.Revision
	cfg := before.Draft
	cfg.Basic.Name = "schema8-release-check-fresh"
	raw, _ := json.Marshal(struct {
		Expected string `json:"expected_draft_revision"`
		app.Config
	}{stale, cfg})
	req, _ := http.NewRequest("PUT", f.host.URL+"/api/v1/cognitive/apps/"+before.ID, bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer demo-engineer")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal("draft save", resp.StatusCode)
	}
	fresh := f.store.Read().Apps[before.ID].Revision
	if fresh == stale {
		t.Fatal("draft revision unchanged")
	}
	staleBody, _ := json.Marshal(map[string]string{"expected_draft_revision": stale})
	for _, address := range []string{f.host.URL, host.URL} {
		req, _ := http.NewRequest("POST", address+"/api/v1/cognitive/apps/"+before.ID+"/release-checks", bytes.NewReader(staleBody))
		req.Header.Set("Authorization", "Bearer demo-engineer")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 409 {
			t.Fatal("stale release-check", address, resp.StatusCode)
		}
	}
	pass := decode(t, f.request("demo-engineer", "POST", "/api/v1/cognitive/apps/"+before.ID+"/release-checks", map[string]string{"expected_draft_revision": fresh}, 200))
	if pass["status"] != "PASS" {
		t.Fatal(pass)
	}
}

// TestR20Schema8DualHostReleasePublishCAS races publish on a single release check.
func TestR20Schema8DualHostReleasePublishCAS(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	appState := f.store.Read().Apps["channel-replenishment"]
	auditsBefore := publishAuditCount(f.store.Read())
	releaseCount := len(appState.Releases)
	check := decode(t, f.request("demo-engineer", "POST", "/api/v1/cognitive/apps/"+appState.ID+"/release-checks", map[string]string{"expected_draft_revision": appState.Revision}, 200))
	if check["status"] != "PASS" {
		t.Fatal(check)
	}
	payload := map[string]any{
		"check_id":                    check["check_id"],
		"expected_draft_revision":     appState.Revision,
		"expected_current_release_id": nil,
		"release_note":                "schema8-dual-publish",
	}
	if appState.CurrentReleaseID != nil {
		payload["expected_current_release_id"] = *appState.CurrentReleaseID
	}
	raw, _ := json.Marshal(payload)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for _, address := range []string{f.host.URL, host.URL} {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			req, _ := http.NewRequest("POST", address+"/api/v1/cognitive/apps/"+appState.ID+"/releases", bytes.NewReader(raw))
			req.Header.Set("Authorization", "Bearer demo-engineer")
			req.Header.Set("Content-Type", "application/json")
			resp, e := http.DefaultClient.Do(req)
			if e != nil {
				t.Error(e)
				results <- 0
				return
			}
			resp.Body.Close()
			results <- resp.StatusCode
		}(address)
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for status := range results {
		counts[status]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal("release publish CAS", counts)
	}
	after := f.store.Read().Apps[appState.ID]
	if len(after.Releases) != releaseCount+1 || publishAuditCount(f.store.Read()) != auditsBefore+1 {
		t.Fatal("duplicate release or audit", len(after.Releases), publishAuditCount(f.store.Read()))
	}
}

// TestR20Schema8DualHostConcurrentGrantRevoke ensures only one revoke audit on concurrent DELETE.
func TestR20Schema8DualHostConcurrentGrantRevoke(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	grantID := "grant-metric-context"
	revokesBefore := countGrantAudit(f.store.Read(), "authorization.revoke")
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for _, address := range []string{f.host.URL, host.URL} {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			req, _ := http.NewRequest("DELETE", address+grantPath+"/"+grantID, nil)
			req.Header.Set("Authorization", "Bearer demo-admin")
			resp, e := http.DefaultClient.Do(req)
			if e != nil {
				t.Error(e)
				results <- 0
				return
			}
			resp.Body.Close()
			results <- resp.StatusCode
		}(address)
	}
	wg.Wait()
	close(results)
	for status := range results {
		if status != 200 {
			t.Fatal("revoke status", status)
		}
	}
	if countGrantAudit(f.store.Read(), "authorization.revoke") != revokesBefore+1 {
		t.Fatal("duplicate revoke audit")
	}
	active := false
	for _, g := range f.store.Read().Grants {
		if g.ID == grantID && g.Active {
			active = true
		}
	}
	if active {
		t.Fatal("grant still active")
	}
}