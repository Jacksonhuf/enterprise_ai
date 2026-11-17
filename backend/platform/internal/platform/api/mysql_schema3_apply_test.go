package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMySQLSchema3ApplyConcurrentSessionRenames(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	seed, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	seed.Close()
	if err = app.MigrateMySQLAudit(context.Background(), dsn); err != nil {
		t.Fatal(err)
	}
	plan, err := app.PlanMySQLApplications(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = app.ApplyMySQLApplications(context.Background(), dsn, plan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	f := setup(t)
	a, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	f.store = a
	f.server.Store = a
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	id := f.session()
	rev := sessionMeta(t, f, id)["revision"].(string)
	raw, _ := json.Marshal(map[string]string{"title": "schema3-concurrent", "expected_revision": rev})
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