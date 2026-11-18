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

// TestR20Schema8DualHostSettingsPublicCAS verifies optimistic concurrency when
// two API hosts concurrently update enterprise public settings (schema8).
func TestR20Schema8DualHostSettingsPublicCAS(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	revision := settingsRead(t, f, "demo-admin", "public")["revision"].(string)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	notices := []string{"schema8-public-A", "schema8-public-B"}
	for i, address := range []string{f.host.URL, host.URL} {
		wg.Add(1)
		go func(i int, address string) {
			defer wg.Done()
			raw, _ := json.Marshal(map[string]any{
				"expected_revision": revision,
				"values":            map[string]string{"notice": notices[i]},
			})
			req, _ := http.NewRequest("PUT", address+settingsBase+"public", bytes.NewReader(raw))
			req.Header.Set("Authorization", "Bearer demo-admin")
			req.Header.Set("Content-Type", "application/json")
			resp, e := http.DefaultClient.Do(req)
			if e != nil {
				t.Error(e)
				results <- 0
				return
			}
			resp.Body.Close()
			results <- resp.StatusCode
		}(i, address)
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for status := range results {
		counts[status]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal("schema8 public settings CAS", counts)
	}
	after := settingsRead(t, f, "demo-admin", "public")
	got := after["values"].(map[string]any)["notice"].(string)
	if got != notices[0] && got != notices[1] {
		t.Fatal("unexpected winning notice", got)
	}
	if settingsAudit(f.store.Read()) < 1 {
		t.Fatal("expected settings audit for public CAS")
	}
}

func dualHostSettingsCAS(t *testing.T, f *fixture, dsn string, actor, section string, values []map[string]string, auditAtLeast int) {
	t.Helper()
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	revision := settingsRead(t, f, actor, section)["revision"].(string)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for i, address := range []string{f.host.URL, host.URL} {
		wg.Add(1)
		go func(i int, address string) {
			defer wg.Done()
			raw, _ := json.Marshal(map[string]any{
				"expected_revision": revision,
				"values":            values[i],
			})
			req, _ := http.NewRequest("PUT", address+settingsBase+section, bytes.NewReader(raw))
			req.Header.Set("Authorization", "Bearer "+actor)
			req.Header.Set("Content-Type", "application/json")
			resp, e := http.DefaultClient.Do(req)
			if e != nil {
				t.Error(e)
				results <- 0
				return
			}
			resp.Body.Close()
			results <- resp.StatusCode
		}(i, address)
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for status := range results {
		counts[status]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal(section, "settings CAS", counts)
	}
	if settingsAudit(f.store.Read()) < auditAtLeast {
		t.Fatal(section, "expected settings audit")
	}
}

// TestR20Schema8DualHostSettingsProfileAndGeneralCAS covers personal settings sections
// on schema8 (complements public CAS and existing profile-only probe).
func TestR20Schema8DualHostSettingsProfileAndGeneralCAS(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	dualHostSettingsCAS(t, f, dsn, "demo-user", "profile",
		[]map[string]string{
			{"display_name": "schema8-profile-A"},
			{"display_name": "schema8-profile-B"},
		}, 1)
	dualHostSettingsCAS(t, f, dsn, "demo-engineer", "general",
		[]map[string]string{
			{"language": "zh-CN", "timezone": "Asia/Shanghai", "density": "compact"},
			{"language": "zh-CN", "timezone": "Asia/Shanghai", "density": "comfortable"},
		}, 2)
}