package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func mysqlSchema8Fixture(t *testing.T) (*fixture, string, *app.Store) {
	t.Helper()
	dsn := testsupport.MySQLDSN(t)
	if err := app.PrepareSchema8IntegrationDB(context.Background(), dsn); err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	f := setup(t)
	f.store = store
	f.server.Store = store
	return f, dsn, store
}

func TestR20Schema8DualHostSessionRenameCAS(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	id := f.session()
	rev := sessionMeta(t, f, id)["revision"].(string)
	raw, _ := json.Marshal(map[string]string{"title": "schema8-http-concurrent", "expected_revision": rev})
	type result struct {
		status int
		err    error
	}
	results := make(chan result, 2)
	for _, address := range []string{f.host.URL, host.URL} {
		go func(address string) {
			r, _ := http.NewRequest("PUT", address+"/api/v1/sessions/"+id, bytes.NewReader(raw))
			r.Header.Set("Authorization", "Bearer demo-user")
			r.Header.Set("Content-Type", "application/json")
			response, e := http.DefaultClient.Do(r)
			if e != nil {
				results <- result{err: e}
				return
			}
			response.Body.Close()
			results <- result{status: response.StatusCode}
		}(address)
	}
	wins, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		switch r.status {
		case 200:
			wins++
		case 409:
			conflicts++
		default:
			t.Fatal("unexpected status", r.status)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatal("dual host session CAS", wins, conflicts)
	}
}

func TestR07Schema8GovernanceUsageRelationalPaging(t *testing.T) {
	f, _, _ := mysqlSchema8Fixture(t)
	path := governanceBase + "usage?limit=2&unit=CALL"
	first := decode(t, f.request("demo-admin", "GET", path, nil, 200))
	items := first["items"].([]any)
	if len(items) != 2 {
		t.Fatal("page size", len(items))
	}
	cursor, ok := first["next_cursor"].(string)
	if !ok || cursor == "" {
		t.Fatal("missing cursor", first["next_cursor"])
	}
	second := decode(t, f.request("demo-admin", "GET", path+"&cursor="+url.QueryEscape(cursor), nil, 200))
	if len(second["items"].([]any)) < 1 {
		t.Fatal("second page empty")
	}
	ids := map[string]bool{}
	for _, page := range []map[string]any{first, second} {
		for _, row := range page["items"].([]any) {
			id := row.(map[string]any)["event_id"].(string)
			if ids[id] {
				t.Fatal("duplicate across pages", id)
			}
			ids[id] = true
		}
	}
}

func TestR20Schema8DualHostConcurrentGrants(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	assertConcurrentGrant(t, f, []string{f.host.URL, host.URL})
	if len(b.Read().Grants) != len(f.store.Read().Grants) {
		t.Fatal("second connection did not observe regrant")
	}
}

func TestR20Schema8DualHostSettingsProfileCAS(t *testing.T) {
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	revision := settingsRead(t, f, "demo-user", "profile")["revision"].(string)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for i, address := range []string{f.host.URL, host.URL} {
		wg.Add(1)
		go func(i int, address string) {
			defer wg.Done()
			raw, _ := json.Marshal(map[string]any{
				"expected_revision": revision,
				"values":            map[string]string{"display_name": []string{"schema8-profile-A", "schema8-profile-B"}[i]},
			})
			req, _ := http.NewRequest("PUT", address+settingsBase+"profile", bytes.NewReader(raw))
			req.Header.Set("Authorization", "Bearer demo-user")
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
	if counts[200] != 1 || counts[409] != 1 || settingsAudit(f.store.Read()) != 1 {
		t.Fatal("schema8 settings CAS", counts)
	}
}

func TestR07Schema8RevokedHistoryRedaction(t *testing.T) {
	f, _, _ := mysqlSchema8Fixture(t)
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	f.request("demo-admin", "DELETE", "/api/v1/cognitive/governance/grants/grant-service", nil, 200)
	result := decode(t, f.request("demo-user", "GET", "/api/v1/messages/"+id+"/load?limit=1", nil, 200))
	message := result["data"].([]any)[0].(map[string]any)
	if message["content_redacted"] != true || message["result"] != nil {
		t.Fatal("revoked result leaked", message)
	}
}