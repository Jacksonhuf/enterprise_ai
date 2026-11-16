package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLExceptionHTTPTransitionsAndIsolation(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			dsn := testsupport.MySQLDSN(t)
			store, e := app.OpenMySQL(dsn)
			if e != nil {
				t.Fatal(e)
			}
			defer store.Close()
			if e = store.Update(func(st *app.State) error {
				st.Exceptions = []map[string]any{{"id": "target", "tenant_id": "42", "status": "OPEN", "revision": "r1"}, {"id": "foreign", "tenant_id": "43", "status": "OPEN", "revision": "other"}}
				return nil
			}); e != nil {
				t.Fatal(e)
			}
			if version == 2 {
				if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
					t.Fatal(e)
				}
			}
			f := setup(t)
			f.store = store
			f.server.Store = store
			db, e := sql.Open("mysql", dsn)
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.exceptions[0].future',CAST(? AS JSON),'$.apps."channel-replenishment".future',CAST(? AS JSON))`, `{"n":9007199254740993}`, `{"n":9007199254740993}`); e != nil {
				t.Fatal(e)
			}
			path := "/api/v1/cognitive/governance/exceptions/target/status"
			f.request("demo-user", "POST", path, map[string]string{"expected_revision": "r1", "status": "ACKNOWLEDGED"}, 403)
			raw := f.request("demo-admin", "POST", path, map[string]string{"expected_revision": "r1", "status": "ACKNOWLEDGED"}, 200)
			first := decode(t, raw)
			if first["status"] != "ACKNOWLEDGED" || first["updated_by"] != "demo-admin" || first["revision"] == "r1" {
				t.Fatal("invalid transition response")
			}
			if version == 2 && !bytes.Contains(raw, []byte("9007199254740993")) {
				t.Fatal("response rounded unknown number")
			}
			f.request("demo-admin", "POST", path, map[string]any{"expected_revision": "r1", "status": "RESOLVED"}, 409)
			f.request("demo-admin", "POST", path, map[string]any{"expected_revision": first["revision"], "status": "OPEN"}, 409)
			second := decode(t, f.request("demo-admin", "POST", path, map[string]any{"expected_revision": first["revision"], "status": "RESOLVED"}, 200))
			f.request("demo-admin", "POST", path, map[string]any{"expected_revision": second["revision"], "status": "OPEN"}, 200)
			f.request("demo-admin", "POST", "/api/v1/cognitive/governance/exceptions/foreign/status", map[string]string{"expected_revision": "other", "status": "RESOLVED"}, 404)
			state := store.Read()
			if state.Exceptions[0]["status"] != "OPEN" || state.Exceptions[1]["status"] != "OPEN" {
				t.Fatal("wrong transition scope")
			}
			count := 0
			for _, event := range state.Audit {
				if strings.HasPrefix(event.Action, "exception.") {
					count++
				}
			}
			if count != 3 {
				t.Fatal("wrong successful transition audit count", count)
			}
			if version == 2 {
				var raw string
				if e = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&raw); e != nil || strings.Count(raw, "9007199254740993") != 2 {
					t.Fatal("status flow rewrote unrelated values", e)
				}
			}
		})
	}
}
func TestMySQLExceptionConcurrentHTTPVersionGuard(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	a, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	if e = a.Update(func(st *app.State) error {
		st.Exceptions = []map[string]any{{"id": "race", "tenant_id": "42", "status": "OPEN", "revision": "r1"}}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	f, g := setup(t), setup(t)
	f.store = a
	f.server.Store = a
	g.store = b
	g.server.Store = b
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	errs := make(chan error, 2)
	for i, item := range []*fixture{f, g} {
		wg.Add(1)
		go func(i int, f *fixture) {
			defer wg.Done()
			raw, _ := json.Marshal(map[string]string{"expected_revision": "r1", "status": []string{"ACKNOWLEDGED", "RESOLVED"}[i]})
			req, _ := http.NewRequest("POST", f.host.URL+"/api/v1/cognitive/governance/exceptions/race/status", bytes.NewReader(raw))
			req.Header.Set("Authorization", "Bearer demo-admin")
			req.Header.Set("Content-Type", "application/json")
			response, e := http.DefaultClient.Do(req)
			if e != nil {
				errs <- e
				return
			}
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			statuses <- response.StatusCode
		}(i, item)
	}
	wg.Wait()
	close(statuses)
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	count := map[int]int{}
	for status := range statuses {
		count[status]++
	}
	state := a.Read()
	audits := 0
	for _, event := range state.Audit {
		if strings.HasPrefix(event.Action, "exception.") {
			audits++
		}
	}
	if count[200] != 1 || count[409] != 1 || audits != 1 || state.Exceptions[0]["revision"] == "r1" {
		t.Fatal("CAS lost", count, audits)
	}
}
